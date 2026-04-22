package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	authctx "agentops/internal/auth"
	"agentops/internal/config"
	"agentops/internal/executor"
	executormock "agentops/internal/executor/mock"
	authlogic "agentops/internal/logic/auth"
	"agentops/internal/model"
	"agentops/internal/svc"
	"agentops/internal/types"
)

type l2TestApp struct {
	db      *sql.DB
	svcCtx  *svc.ServiceContext
	repoDir string
}

func newL2TestApp(t *testing.T) *l2TestApp {
	t.Helper()

	repoDir := testRepoRoot(t)
	dbName := fmt.Sprintf("agentops_test_%d_%d", time.Now().UnixNano(), rand.Intn(100000))

	adminCfg := testPostgresConf()
	adminCfg.DBName = getenvDefault("AGENTOPS_TEST_ADMIN_DB", "postgres")

	adminDB, err := model.NewPostgresDB(adminCfg)
	if err != nil {
		t.Fatalf("connect admin db: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName)); err != nil {
		_ = adminDB.Close()
		t.Fatalf("create test db: %v", err)
	}

	testCfg := testPostgresConf()
	testCfg.DBName = dbName

	db, err := model.NewPostgresDB(testCfg)
	if err != nil {
		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		_ = adminDB.Close()
		t.Fatalf("connect test db: %v", err)
	}

	applyMigrations(t, db, repoDir)

	app := &l2TestApp{
		db:      db,
		repoDir: repoDir,
		svcCtx: &svc.ServiceContext{
			Config: config.Config{
				Auth: config.AuthConf{
					AccessSecret: "agentops-l3-test-secret",
					AccessExpire: 3600,
				},
				Executor: config.ExecutorConf{
					TimeoutSeconds:   30,
					AllowedRepoPaths: []string{repoDir},
				},
			},
			DB:                     db,
			UserModel:              model.NewUserModel(db),
			TaskModel:              model.NewTaskModel(db),
			TaskPolicyModel:        model.NewTaskPolicyModel(db),
			AuditLogModel:          model.NewAuditLogModel(db),
			ApprovalRecordModel:    model.NewApprovalRecordModel(db),
			TaskExecutionModel:     model.NewTaskExecutionModel(db),
			TaskStatusHistoryModel: model.NewTaskStatusHistoryModel(db),
			TaskRunner:             executormock.NewRunner(),
		},
	}

	t.Cleanup(func() {
		_ = db.Close()

		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()

		_, _ = adminDB.ExecContext(dropCtx, "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()", dbName)
		_, _ = adminDB.ExecContext(dropCtx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		_ = adminDB.Close()
	})

	return app
}

func TestL3LoginSuccessAndFailure(t *testing.T) {
	app := newL2TestApp(t)

	loginLogic := authlogic.NewLoginLogic(context.Background(), app.svcCtx)
	resp, err := loginLogic.Login(&types.LoginReq{
		Username: "creator-1",
		Password: "password",
	})
	if err != nil {
		t.Fatalf("login success: %v", err)
	}
	if resp.AccessToken == "" || resp.ExpiresIn != 3600 {
		t.Fatalf("unexpected login response: %+v", resp)
	}

	if _, err := loginLogic.Login(&types.LoginReq{
		Username: "creator-1",
		Password: "wrong",
	}); !errors.Is(err, authlogic.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestL2CreateTaskAndQueryStatusHistory(t *testing.T) {
	app := newL2TestApp(t)

	creatorCtx := testActorContext("creator-1", authctx.SystemRoleViewer)
	createLogic := NewCreateTaskLogic(creatorCtx, app.svcCtx)
	resp, err := createLogic.CreateTask(&types.CreateTaskReq{
		Title:            "l2 create history",
		RepoPath:         app.repoDir,
		Prompt:           "trace status history",
		ReviewerId:       "reviewer-1",
		OperatorId:       "operator-1",
		Mode:             TaskModeAnalyze,
		ApprovalRequired: true,
		MaxSteps:         3,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if resp.Status != TaskStatusWaitingApproval {
		t.Fatalf("unexpected create status: %s", resp.Status)
	}

	historyLogic := NewGetTaskStatusHistoriesLogic(creatorCtx, app.svcCtx)
	historyResp, err := historyLogic.GetTaskStatusHistories(&types.GetTaskStatusHistoriesReq{Id: resp.Id})
	if err != nil {
		t.Fatalf("get status histories: %v", err)
	}
	if len(historyResp.Items) != 1 {
		t.Fatalf("expected 1 status history, got %d", len(historyResp.Items))
	}

	item := historyResp.Items[0]
	if item.FromStatus != "" || item.ToStatus != TaskStatusWaitingApproval || item.Action != "create" {
		t.Fatalf("unexpected status history item: %+v", item)
	}
	if item.ActorId != "creator-1" || item.ActorRole != "creator" {
		t.Fatalf("unexpected actor info: %+v", item)
	}
}

func TestL2CreateTaskRejectsNonGitRepo(t *testing.T) {
	app := newL2TestApp(t)

	nonGitDir := t.TempDir()
	createLogic := NewCreateTaskLogic(testActorContext("creator-1", authctx.SystemRoleViewer), app.svcCtx)
	_, err := createLogic.CreateTask(&types.CreateTaskReq{
		Title:            "not git",
		RepoPath:         nonGitDir,
		Prompt:           "should fail",
		Mode:             TaskModeAnalyze,
		ApprovalRequired: false,
		MaxSteps:         1,
	})
	if !errors.Is(err, ErrRepoNotGitRepo) {
		t.Fatalf("expected ErrRepoNotGitRepo, got %v", err)
	}
}

func TestL2ApproveStartSucceedFlowAndRepeatGuards(t *testing.T) {
	app := newL2TestApp(t)

	taskID := createTaskForTest(t, app, types.CreateTaskReq{
		Title:            "approve-start-succeed",
		RepoPath:         app.repoDir,
		Prompt:           "full happy path",
		ReviewerId:       "reviewer-1",
		OperatorId:       "operator-1",
		Mode:             TaskModePatch,
		ApprovalRequired: true,
		MaxSteps:         5,
		AllowedPaths:     []string{"internal/logic/tasks"},
	})

	if _, err := NewApproveTaskLogic(testActorContext("creator-1", authctx.SystemRoleViewer), app.svcCtx).ApproveTask(&types.ApproveTaskReq{
		Id: taskID,
	}); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
	if _, err := NewApproveTaskLogic(testActorContext("reviewer-2", authctx.SystemRoleReviewer), app.svcCtx).ApproveTask(&types.ApproveTaskReq{
		Id: taskID,
	}); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
	approveLogic := NewApproveTaskLogic(testActorContext("reviewer-1", authctx.SystemRoleReviewer), app.svcCtx)
	if _, err := approveLogic.ApproveTask(&types.ApproveTaskReq{
		Id:     taskID,
		Reason: "approved",
	}); err != nil {
		t.Fatalf("approve task: %v", err)
	}
	if _, err := approveLogic.ApproveTask(&types.ApproveTaskReq{
		Id: taskID,
	}); !errors.Is(err, ErrTaskNotWaitingApproval) {
		t.Fatalf("expected ErrTaskNotWaitingApproval, got %v", err)
	}

	startLogic := NewStartTaskLogic(testActorContext("operator-2", authctx.SystemRoleOperator), app.svcCtx)
	if _, err := startLogic.StartTask(&types.StartTaskReq{
		Id: taskID,
	}); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
	startLogic = NewStartTaskLogic(testActorContext("operator-1", authctx.SystemRoleOperator), app.svcCtx)
	startResp, err := startLogic.StartTask(&types.StartTaskReq{
		Id: taskID,
	})
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if startResp.Status != TaskStatusSucceeded {
		t.Fatalf("expected synced start to succeed, got %s", startResp.Status)
	}
	if _, err := startLogic.StartTask(&types.StartTaskReq{
		Id: taskID,
	}); !errors.Is(err, ErrTaskNotPending) {
		t.Fatalf("expected ErrTaskNotPending, got %v", err)
	}

	succeedLogic := NewSucceedTaskLogic(testActorContext("operator-2", authctx.SystemRoleOperator), app.svcCtx)
	if _, err := succeedLogic.SucceedTask(&types.SucceedTaskReq{
		Id: taskID,
	}); !errors.Is(err, ErrTaskNotRunning) {
		t.Fatalf("expected ErrTaskNotRunning, got %v", err)
	}

	execLogic := NewGetTaskExecutionsLogic(testActorContext("operator-1", authctx.SystemRoleOperator), app.svcCtx)
	execResp, err := execLogic.GetTaskExecutions(&types.GetTaskExecutionsReq{Id: taskID})
	if err != nil {
		t.Fatalf("get executions: %v", err)
	}
	if len(execResp.Items) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(execResp.Items))
	}
	if execResp.Items[0].Status != TaskStatusSucceeded ||
		execResp.Items[0].OperatorId != "operator-1" ||
		!strings.Contains(execResp.Items[0].ResultSummary, "summary: mock executor completed task") {
		t.Fatalf("unexpected execution item: %+v", execResp.Items[0])
	}

	historyLogic := NewGetTaskStatusHistoriesLogic(testActorContext("creator-1", authctx.SystemRoleViewer), app.svcCtx)
	historyResp, err := historyLogic.GetTaskStatusHistories(&types.GetTaskStatusHistoriesReq{Id: taskID})
	if err != nil {
		t.Fatalf("get status histories: %v", err)
	}
	if len(historyResp.Items) != 4 {
		t.Fatalf("expected 4 histories, got %d", len(historyResp.Items))
	}

	actions := []string{
		historyResp.Items[0].Action,
		historyResp.Items[1].Action,
		historyResp.Items[2].Action,
		historyResp.Items[3].Action,
	}
	expectedActions := []string{"create", "approve", "start", "succeed"}
	if strings.Join(actions, ",") != strings.Join(expectedActions, ",") {
		t.Fatalf("unexpected history actions: %v", actions)
	}
}

func TestL2FailAndCancelBehaviors(t *testing.T) {
	app := newL2TestApp(t)

	pendingTaskID := createTaskForTest(t, app, types.CreateTaskReq{
		Title:            "cancel pending",
		RepoPath:         app.repoDir,
		Prompt:           "cancel",
		Mode:             TaskModeAnalyze,
		ApprovalRequired: false,
		MaxSteps:         2,
	})

	cancelLogic := NewCancelTaskLogic(testActorContext("creator-2", authctx.SystemRoleViewer), app.svcCtx)
	if _, err := cancelLogic.CancelTask(&types.CancelTaskReq{
		Id: pendingTaskID,
	}); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}

	cancelLogic = NewCancelTaskLogic(testActorContext("creator-1", authctx.SystemRoleViewer), app.svcCtx)
	if _, err := cancelLogic.CancelTask(&types.CancelTaskReq{
		Id:     pendingTaskID,
		Reason: "stop here",
	}); err != nil {
		t.Fatalf("cancel task: %v", err)
	}

	getLogic := NewGetTaskLogic(testActorContext("creator-1", authctx.SystemRoleViewer), app.svcCtx)
	taskResp, err := getLogic.GetTask(&types.GetTaskReq{Id: pendingTaskID})
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if taskResp.Status != TaskStatusCancelled || taskResp.CancelledBy != "creator-1" || taskResp.CancelledAt == "" {
		t.Fatalf("unexpected cancelled task detail: %+v", taskResp)
	}

	runningTaskID := createTaskForTest(t, app, types.CreateTaskReq{
		Title:            "fail running",
		RepoPath:         app.repoDir,
		Prompt:           "fail path",
		OperatorId:       "operator-1",
		Mode:             TaskModePatch,
		ApprovalRequired: false,
		MaxSteps:         2,
		AllowedPaths:     []string{"internal/logic/tasks"},
	})

	app.svcCtx.TaskRunner = failingTestRunner{err: errors.New("boom")}
	startLogic := NewStartTaskLogic(testActorContext("operator-1", authctx.SystemRoleOperator), app.svcCtx)
	startResp, err := startLogic.StartTask(&types.StartTaskReq{
		Id: runningTaskID,
	})
	if err != nil {
		t.Fatalf("start running task: %v", err)
	}
	if startResp.Status != TaskStatusFailed {
		t.Fatalf("expected synced start to fail, got %s", startResp.Status)
	}

	if _, err := cancelLogic.CancelTask(&types.CancelTaskReq{
		Id: runningTaskID,
	}); !errors.Is(err, ErrTaskCannotBeCancelled) {
		t.Fatalf("expected ErrTaskCannotBeCancelled, got %v", err)
	}

	failLogic := NewFailTaskLogic(testActorContext("operator-1", authctx.SystemRoleOperator), app.svcCtx)
	if _, err := failLogic.FailTask(&types.FailTaskReq{
		Id: runningTaskID,
	}); !errors.Is(err, ErrErrorMessageRequired) {
		t.Fatalf("expected ErrErrorMessageRequired, got %v", err)
	}
	if _, err := NewFailTaskLogic(testActorContext("operator-2", authctx.SystemRoleOperator), app.svcCtx).FailTask(&types.FailTaskReq{
		Id:            runningTaskID,
		ResultSummary: "partial",
		ErrorMessage:  "boom",
	}); !errors.Is(err, ErrTaskNotRunning) {
		t.Fatalf("expected ErrTaskNotRunning, got %v", err)
	}
	if _, err := failLogic.FailTask(&types.FailTaskReq{
		Id:           runningTaskID,
		ErrorMessage: "boom again",
	}); !errors.Is(err, ErrTaskNotRunning) {
		t.Fatalf("expected ErrTaskNotRunning, got %v", err)
	}

	execLogic := NewGetTaskExecutionsLogic(testActorContext("operator-1", authctx.SystemRoleOperator), app.svcCtx)
	execResp, err := execLogic.GetTaskExecutions(&types.GetTaskExecutionsReq{Id: runningTaskID})
	if err != nil {
		t.Fatalf("get executions after fail: %v", err)
	}
	if len(execResp.Items) != 1 {
		t.Fatalf("expected 1 failed execution, got %d", len(execResp.Items))
	}
	if execResp.Items[0].Status != TaskStatusFailed || execResp.Items[0].ErrorMessage != "boom" {
		t.Fatalf("unexpected failed execution: %+v", execResp.Items[0])
	}
}

func TestL4StartFailsWhenExecutorWritesOutsideAllowedPaths(t *testing.T) {
	app := newL2TestApp(t)

	outsidePath := fmt.Sprintf("l4-outside-%d.txt", time.Now().UnixNano())
	t.Cleanup(func() {
		_ = os.Remove(filepath.Join(app.repoDir, outsidePath))
	})

	taskID := createTaskForTest(t, app, types.CreateTaskReq{
		Title:            "deny outside path",
		RepoPath:         app.repoDir,
		Prompt:           "write outside allowed path",
		OperatorId:       "operator-1",
		Mode:             TaskModePatch,
		ApprovalRequired: false,
		MaxSteps:         2,
		AllowedPaths:     []string{"internal/logic/tasks"},
	})

	app.svcCtx.TaskRunner = writingTestRunner{
		Path:    outsidePath,
		Content: "outside allowed path",
	}

	startResp, err := NewStartTaskLogic(testActorContext("operator-1", authctx.SystemRoleOperator), app.svcCtx).StartTask(&types.StartTaskReq{
		Id: taskID,
	})
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if startResp.Status != TaskStatusFailed {
		t.Fatalf("expected synced start to fail, got %s", startResp.Status)
	}

	execResp, err := NewGetTaskExecutionsLogic(testActorContext("operator-1", authctx.SystemRoleOperator), app.svcCtx).GetTaskExecutions(&types.GetTaskExecutionsReq{Id: taskID})
	if err != nil {
		t.Fatalf("get executions: %v", err)
	}
	if len(execResp.Items) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(execResp.Items))
	}
	if execResp.Items[0].Status != TaskStatusFailed ||
		!strings.Contains(execResp.Items[0].ErrorMessage, ErrChangedFileNotAllowed.Error()) ||
		!strings.Contains(execResp.Items[0].ErrorMessage, outsidePath) ||
		!strings.Contains(execResp.Items[0].ResultSummary, "gitNewChangedFiles: ["+outsidePath+"]") {
		t.Fatalf("unexpected failed execution: %+v", execResp.Items[0])
	}
}

func TestL2ConcurrentApproveOnlyOneSucceeds(t *testing.T) {
	app := newL2TestApp(t)

	taskID := createTaskForTest(t, app, types.CreateTaskReq{
		Title:            "concurrent approve",
		RepoPath:         app.repoDir,
		Prompt:           "approve race",
		ReviewerId:       "reviewer-1",
		Mode:             TaskModeAnalyze,
		ApprovalRequired: true,
		MaxSteps:         2,
	})

	errs := runConcurrently(4, func() error {
		_, err := NewApproveTaskLogic(testActorContext("reviewer-1", authctx.SystemRoleReviewer), app.svcCtx).ApproveTask(&types.ApproveTaskReq{
			Id: taskID,
		})
		return err
	})

	assertConcurrencyOutcome(t, errs, ErrTaskNotWaitingApproval)
}

func TestL2ConcurrentStartOnlyOneSucceeds(t *testing.T) {
	app := newL2TestApp(t)

	taskID := createTaskForTest(t, app, types.CreateTaskReq{
		Title:            "concurrent start",
		RepoPath:         app.repoDir,
		Prompt:           "start race",
		OperatorId:       "operator-1",
		Mode:             TaskModePatch,
		ApprovalRequired: false,
		MaxSteps:         2,
		AllowedPaths:     []string{"internal/logic/tasks"},
	})

	errs := runConcurrently(4, func() error {
		_, err := NewStartTaskLogic(testActorContext("operator-1", authctx.SystemRoleOperator), app.svcCtx).StartTask(&types.StartTaskReq{
			Id: taskID,
		})
		return err
	})

	assertConcurrencyOutcome(t, errs, ErrTaskNotPending)
}

func createTaskForTest(t *testing.T, app *l2TestApp, req types.CreateTaskReq) string {
	t.Helper()

	resp, err := NewCreateTaskLogic(testActorContext("creator-1", authctx.SystemRoleViewer), app.svcCtx).CreateTask(&req)
	if err != nil {
		t.Fatalf("create task for test: %v", err)
	}

	return resp.Id
}

func testActorContext(id string, role string) context.Context {
	return authctx.WithCurrentUser(context.Background(), authctx.CurrentUser{
		ID:         id,
		Username:   id,
		SystemRole: role,
	})
}

type failingTestRunner struct {
	err error
}

func (r failingTestRunner) Run(ctx context.Context, req executor.Request) (executor.Result, error) {
	return executor.Result{
		Summary: "test runner failed",
	}, r.err
}

type writingTestRunner struct {
	Path    string
	Content string
}

func (r writingTestRunner) Run(ctx context.Context, req executor.Request) (executor.Result, error) {
	fullPath := filepath.Join(req.Repo.Path, r.Path)
	if err := os.WriteFile(fullPath, []byte(r.Content), 0644); err != nil {
		return executor.Result{
			Summary: "write test file failed",
			Stderr:  err.Error(),
		}, err
	}

	return executor.Result{
		Summary: "write test file completed",
		Stdout:  r.Path,
	}, nil
}

func runConcurrently(n int, fn func() error) []error {
	start := make(chan struct{})
	errs := make([]error, n)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			errs[idx] = fn()
		}(i)
	}

	close(start)
	wg.Wait()

	return errs
}

func assertConcurrencyOutcome(t *testing.T, errs []error, expectedErr error) {
	t.Helper()

	successes := 0
	expectedFailures := 0

	for _, err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, expectedErr):
			expectedFailures++
		default:
			t.Fatalf("unexpected concurrent error: %v", err)
		}
	}

	if successes != 1 {
		t.Fatalf("expected exactly 1 success, got %d (errs=%v)", successes, errs)
	}
	if expectedFailures != len(errs)-1 {
		t.Fatalf("expected %d failures with %v, got %d", len(errs)-1, expectedErr, expectedFailures)
	}
}

func applyMigrations(t *testing.T, db *sql.DB, repoDir string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, name := range []string{
		"0001_init.sql",
		"0003_l3_auth_and_visibility.sql",
	} {
		content, err := os.ReadFile(filepath.Join(repoDir, "migrations", name))
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}

		statements := strings.Split(string(content), ";")
		for _, statement := range statements {
			stmt := strings.TrimSpace(statement)
			if stmt == "" {
				continue
			}
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				t.Fatalf("apply migration %s statement %q: %v", name, stmt, err)
			}
		}
	}

	seedL3Users(t, db)
}

func seedL3Users(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	passwordHash, err := authctx.HashPassword("password")
	if err != nil {
		t.Fatalf("hash test password: %v", err)
	}

	users := []model.User{
		{ID: "admin-1", Username: "admin-1", PasswordHash: passwordHash, SystemRole: authctx.SystemRoleAdmin},
		{ID: "creator-1", Username: "creator-1", PasswordHash: passwordHash, SystemRole: authctx.SystemRoleViewer},
		{ID: "creator-2", Username: "creator-2", PasswordHash: passwordHash, SystemRole: authctx.SystemRoleViewer},
		{ID: "reviewer-1", Username: "reviewer-1", PasswordHash: passwordHash, SystemRole: authctx.SystemRoleReviewer},
		{ID: "reviewer-2", Username: "reviewer-2", PasswordHash: passwordHash, SystemRole: authctx.SystemRoleReviewer},
		{ID: "operator-1", Username: "operator-1", PasswordHash: passwordHash, SystemRole: authctx.SystemRoleOperator},
		{ID: "operator-2", Username: "operator-2", PasswordHash: passwordHash, SystemRole: authctx.SystemRoleOperator},
	}

	userModel := model.NewUserModel(db)
	for _, user := range users {
		if err := userModel.Insert(ctx, db, &user); err != nil {
			t.Fatalf("seed user %s: %v", user.ID, err)
		}
	}
}

func testRepoRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test filename")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func testPostgresConf() config.PostgresConf {
	return config.PostgresConf{
		Host:                   getenvDefault("AGENTOPS_TEST_PG_HOST", "127.0.0.1"),
		Port:                   mustAtoi(getenvDefault("AGENTOPS_TEST_PG_PORT", "5432")),
		User:                   getenvDefault("AGENTOPS_TEST_PG_USER", "postgres"),
		Password:               getenvDefault("AGENTOPS_TEST_PG_PASSWORD", "postgres"),
		DBName:                 getenvDefault("AGENTOPS_TEST_PG_DB", "agentops"),
		SSLMode:                getenvDefault("AGENTOPS_TEST_PG_SSLMODE", "disable"),
		MaxOpenConns:           10,
		MaxIdleConns:           5,
		ConnMaxLifetimeSeconds: 600,
	}
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func mustAtoi(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		panic(err)
	}
	return n
}
