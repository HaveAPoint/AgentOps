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

	"agentops/internal/config"
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
			DB:                     db,
			TaskModel:              model.NewTaskModel(db),
			TaskPolicyModel:        model.NewTaskPolicyModel(db),
			AuditLogModel:          model.NewAuditLogModel(db),
			ApprovalRecordModel:    model.NewApprovalRecordModel(db),
			TaskExecutionModel:     model.NewTaskExecutionModel(db),
			TaskStatusHistoryModel: model.NewTaskStatusHistoryModel(db),
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

func TestL2CreateTaskAndQueryStatusHistory(t *testing.T) {
	app := newL2TestApp(t)

	createLogic := NewCreateTaskLogic(context.Background(), app.svcCtx)
	resp, err := createLogic.CreateTask(&types.CreateTaskReq{
		Title:            "l2 create history",
		RepoPath:         app.repoDir,
		Prompt:           "trace status history",
		CreatorId:        "creator-1",
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

	historyLogic := NewGetTaskStatusHistoriesLogic(context.Background(), app.svcCtx)
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
	createLogic := NewCreateTaskLogic(context.Background(), app.svcCtx)
	_, err := createLogic.CreateTask(&types.CreateTaskReq{
		Title:            "not git",
		RepoPath:         nonGitDir,
		Prompt:           "should fail",
		CreatorId:        "creator-1",
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
		CreatorId:        "creator-1",
		ReviewerId:       "reviewer-1",
		OperatorId:       "operator-1",
		Mode:             TaskModePatch,
		ApprovalRequired: true,
		MaxSteps:         5,
	})

	approveLogic := NewApproveTaskLogic(context.Background(), app.svcCtx)
	if _, err := approveLogic.ApproveTask(&types.ApproveTaskReq{
		Id:         taskID,
		ReviewerId: "",
	}); !errors.Is(err, ErrReviewerIDRequired) {
		t.Fatalf("expected ErrReviewerIDRequired, got %v", err)
	}
	if _, err := approveLogic.ApproveTask(&types.ApproveTaskReq{
		Id:         taskID,
		ReviewerId: "reviewer-2",
	}); !errors.Is(err, ErrReviewerIDMismatch) {
		t.Fatalf("expected ErrReviewerIDMismatch, got %v", err)
	}
	if _, err := approveLogic.ApproveTask(&types.ApproveTaskReq{
		Id:         taskID,
		ReviewerId: "reviewer-1",
		Reason:     "approved",
	}); err != nil {
		t.Fatalf("approve task: %v", err)
	}
	if _, err := approveLogic.ApproveTask(&types.ApproveTaskReq{
		Id:         taskID,
		ReviewerId: "reviewer-1",
	}); !errors.Is(err, ErrTaskNotWaitingApproval) {
		t.Fatalf("expected ErrTaskNotWaitingApproval, got %v", err)
	}

	startLogic := NewStartTaskLogic(context.Background(), app.svcCtx)
	if _, err := startLogic.StartTask(&types.StartTaskReq{
		Id:         taskID,
		OperatorId: "operator-2",
	}); !errors.Is(err, ErrOperatorIDMismatch) {
		t.Fatalf("expected ErrOperatorIDMismatch, got %v", err)
	}
	if _, err := startLogic.StartTask(&types.StartTaskReq{
		Id:         taskID,
		OperatorId: "operator-1",
	}); err != nil {
		t.Fatalf("start task: %v", err)
	}
	if _, err := startLogic.StartTask(&types.StartTaskReq{
		Id:         taskID,
		OperatorId: "operator-1",
	}); !errors.Is(err, ErrTaskNotPending) {
		t.Fatalf("expected ErrTaskNotPending, got %v", err)
	}

	succeedLogic := NewSucceedTaskLogic(context.Background(), app.svcCtx)
	if _, err := succeedLogic.SucceedTask(&types.SucceedTaskReq{
		Id:         taskID,
		OperatorId: "operator-2",
	}); !errors.Is(err, ErrOperatorIDMismatch) {
		t.Fatalf("expected ErrOperatorIDMismatch, got %v", err)
	}
	if _, err := succeedLogic.SucceedTask(&types.SucceedTaskReq{
		Id:            taskID,
		OperatorId:    "operator-1",
		ResultSummary: "done",
	}); err != nil {
		t.Fatalf("succeed task: %v", err)
	}
	if _, err := succeedLogic.SucceedTask(&types.SucceedTaskReq{
		Id:         taskID,
		OperatorId: "operator-1",
	}); !errors.Is(err, ErrTaskNotRunning) {
		t.Fatalf("expected ErrTaskNotRunning, got %v", err)
	}

	execLogic := NewGetTaskExecutionsLogic(context.Background(), app.svcCtx)
	execResp, err := execLogic.GetTaskExecutions(&types.GetTaskExecutionsReq{Id: taskID})
	if err != nil {
		t.Fatalf("get executions: %v", err)
	}
	if len(execResp.Items) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(execResp.Items))
	}
	if execResp.Items[0].Status != TaskStatusSucceeded || execResp.Items[0].OperatorId != "operator-1" || execResp.Items[0].ResultSummary != "done" {
		t.Fatalf("unexpected execution item: %+v", execResp.Items[0])
	}

	historyLogic := NewGetTaskStatusHistoriesLogic(context.Background(), app.svcCtx)
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
		CreatorId:        "creator-1",
		Mode:             TaskModeAnalyze,
		ApprovalRequired: false,
		MaxSteps:         2,
	})

	cancelLogic := NewCancelTaskLogic(context.Background(), app.svcCtx)
	if _, err := cancelLogic.CancelTask(&types.CancelTaskReq{
		Id:        pendingTaskID,
		ActorId:   "creator-2",
		ActorRole: "creator",
	}); !errors.Is(err, ErrCancelActorNotAllowed) {
		t.Fatalf("expected ErrCancelActorNotAllowed, got %v", err)
	}

	if _, err := cancelLogic.CancelTask(&types.CancelTaskReq{
		Id:        pendingTaskID,
		ActorId:   "creator-1",
		ActorRole: "creator",
		Reason:    "stop here",
	}); err != nil {
		t.Fatalf("cancel task: %v", err)
	}

	getLogic := NewGetTaskLogic(context.Background(), app.svcCtx)
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
		CreatorId:        "creator-1",
		OperatorId:       "operator-1",
		Mode:             TaskModePatch,
		ApprovalRequired: false,
		MaxSteps:         2,
	})

	startLogic := NewStartTaskLogic(context.Background(), app.svcCtx)
	if _, err := startLogic.StartTask(&types.StartTaskReq{
		Id:         runningTaskID,
		OperatorId: "operator-1",
	}); err != nil {
		t.Fatalf("start running task: %v", err)
	}

	if _, err := cancelLogic.CancelTask(&types.CancelTaskReq{
		Id:        runningTaskID,
		ActorId:   "creator-1",
		ActorRole: "creator",
	}); !errors.Is(err, ErrTaskCannotBeCancelled) {
		t.Fatalf("expected ErrTaskCannotBeCancelled, got %v", err)
	}

	failLogic := NewFailTaskLogic(context.Background(), app.svcCtx)
	if _, err := failLogic.FailTask(&types.FailTaskReq{
		Id:         runningTaskID,
		OperatorId: "operator-1",
	}); !errors.Is(err, ErrErrorMessageRequired) {
		t.Fatalf("expected ErrErrorMessageRequired, got %v", err)
	}
	if _, err := failLogic.FailTask(&types.FailTaskReq{
		Id:            runningTaskID,
		OperatorId:    "operator-2",
		ResultSummary: "partial",
		ErrorMessage:  "boom",
	}); !errors.Is(err, ErrOperatorIDMismatch) {
		t.Fatalf("expected ErrOperatorIDMismatch, got %v", err)
	}
	if _, err := failLogic.FailTask(&types.FailTaskReq{
		Id:            runningTaskID,
		OperatorId:    "operator-1",
		ResultSummary: "partial",
		ErrorMessage:  "boom",
	}); err != nil {
		t.Fatalf("fail task: %v", err)
	}
	if _, err := failLogic.FailTask(&types.FailTaskReq{
		Id:           runningTaskID,
		OperatorId:   "operator-1",
		ErrorMessage: "boom again",
	}); !errors.Is(err, ErrTaskNotRunning) {
		t.Fatalf("expected ErrTaskNotRunning, got %v", err)
	}

	execLogic := NewGetTaskExecutionsLogic(context.Background(), app.svcCtx)
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

func TestL2ConcurrentApproveOnlyOneSucceeds(t *testing.T) {
	app := newL2TestApp(t)

	taskID := createTaskForTest(t, app, types.CreateTaskReq{
		Title:            "concurrent approve",
		RepoPath:         app.repoDir,
		Prompt:           "approve race",
		CreatorId:        "creator-1",
		ReviewerId:       "reviewer-1",
		Mode:             TaskModeAnalyze,
		ApprovalRequired: true,
		MaxSteps:         2,
	})

	errs := runConcurrently(4, func() error {
		_, err := NewApproveTaskLogic(context.Background(), app.svcCtx).ApproveTask(&types.ApproveTaskReq{
			Id:         taskID,
			ReviewerId: "reviewer-1",
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
		CreatorId:        "creator-1",
		OperatorId:       "operator-1",
		Mode:             TaskModePatch,
		ApprovalRequired: false,
		MaxSteps:         2,
	})

	errs := runConcurrently(4, func() error {
		_, err := NewStartTaskLogic(context.Background(), app.svcCtx).StartTask(&types.StartTaskReq{
			Id:         taskID,
			OperatorId: "operator-1",
		})
		return err
	})

	assertConcurrencyOutcome(t, errs, ErrTaskNotPending)
}

func createTaskForTest(t *testing.T, app *l2TestApp, req types.CreateTaskReq) string {
	t.Helper()

	resp, err := NewCreateTaskLogic(context.Background(), app.svcCtx).CreateTask(&req)
	if err != nil {
		t.Fatalf("create task for test: %v", err)
	}

	return resp.Id
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

	content, err := os.ReadFile(filepath.Join(repoDir, "migrations", "0001_init.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	statements := strings.Split(string(content), ";")
	for _, statement := range statements {
		stmt := strings.TrimSpace(statement)
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("apply migration statement %q: %v", stmt, err)
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
