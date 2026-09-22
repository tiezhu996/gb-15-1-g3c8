package service

import (
	"context"
	"testing"

	"github.com/paperflow/paperflow/internal/constants"
	"github.com/paperflow/paperflow/internal/dto"
	"github.com/paperflow/paperflow/internal/model"
	"github.com/paperflow/paperflow/internal/util"
)

func newWithdrawalTestEnv(t *testing.T) (*fakeStore, *WithdrawalService) {
	t.Helper()
	store := newFakeStore()
	svc := NewWithdrawalService(store, newTestLogger())
	return store, svc
}

func seedPaperForWithdrawal(t *testing.T, store *fakeStore, status string, submitterID uint) *model.Paper {
	t.Helper()
	p := &model.Paper{
		Title: "撤稿测试论文-" + status, Abstract: "摘要", Keywords: "撤稿", Subject: "computer",
		Status: status, SubmitterID: submitterID,
	}
	if err := store.papers.Create(context.Background(), p); err != nil {
		t.Fatalf("seed paper: %v", err)
	}
	return p
}

func TestWithdrawalServiceApply(t *testing.T) {
	ctx := context.Background()

	t.Run("submitted paper applies without alt note", func(t *testing.T) {
		store, svc := newWithdrawalTestEnv(t)
		paper := seedPaperForWithdrawal(t, store, constants.PaperStatusSubmitted, 1)
		w, err := svc.Apply(ctx, 1, paper.ID, dto.CreateWithdrawalRequest{Reason: "作者主动撤稿：研究方向调整"})
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		if w.Status != constants.WithdrawalStatusPending {
			t.Errorf("expected pending, got %s", w.Status)
		}
	})

	t.Run("external review requires alt handling note", func(t *testing.T) {
		store, svc := newWithdrawalTestEnv(t)
		paper := seedPaperForWithdrawal(t, store, constants.PaperStatusExternalReview, 1)
		if _, err := svc.Apply(ctx, 1, paper.ID, dto.CreateWithdrawalRequest{Reason: "作者主动撤稿：数据需补充"}); err == nil {
			t.Fatalf("expected error: alt handling note required for external_review")
		}
		w, err := svc.Apply(ctx, 1, paper.ID, dto.CreateWithdrawalRequest{
			Reason: "作者主动撤稿：数据需补充", AltHandlingNote: "已通知审稿人停止审稿，改投其他期刊",
		})
		if err != nil {
			t.Fatalf("apply with alt note: %v", err)
		}
		if w.AltHandlingNote == "" {
			t.Errorf("expected alt handling note persisted")
		}
	})

	t.Run("revision requires alt handling note", func(t *testing.T) {
		store, svc := newWithdrawalTestEnv(t)
		paper := seedPaperForWithdrawal(t, store, constants.PaperStatusRevision, 1)
		if _, err := svc.Apply(ctx, 1, paper.ID, dto.CreateWithdrawalRequest{Reason: "作者主动撤稿：无法按期修回"}); err == nil {
			t.Fatalf("expected error: alt handling note required for revision")
		}
	})

	t.Run("terminal status rejected", func(t *testing.T) {
		store, svc := newWithdrawalTestEnv(t)
		for _, st := range []string{constants.PaperStatusAccepted, constants.PaperStatusRejected, constants.PaperStatusWithdrawn} {
			paper := seedPaperForWithdrawal(t, store, st, 1)
			if _, err := svc.Apply(ctx, 1, paper.ID, dto.CreateWithdrawalRequest{Reason: "作者主动撤稿：终态保护校验"}); err == nil {
				t.Fatalf("expected error for terminal status %s", st)
			}
		}
	})

	t.Run("duplicate application keeps only one", func(t *testing.T) {
		store, svc := newWithdrawalTestEnv(t)
		paper := seedPaperForWithdrawal(t, store, constants.PaperStatusSubmitted, 1)
		if _, err := svc.Apply(ctx, 1, paper.ID, dto.CreateWithdrawalRequest{Reason: "作者主动撤稿：第一次申请"}); err != nil {
			t.Fatalf("first apply: %v", err)
		}
		if _, err := svc.Apply(ctx, 1, paper.ID, dto.CreateWithdrawalRequest{Reason: "作者主动撤稿：重复申请"}); err == nil {
			t.Fatalf("expected error for duplicate pending application")
		}
		items, err := svc.ListByPaper(ctx, paper.ID)
		if err != nil {
			t.Fatalf("list by paper: %v", err)
		}
		if len(items) != 1 {
			t.Errorf("expected exactly 1 withdrawal record, got %d", len(items))
		}
	})

	t.Run("non owner cannot apply", func(t *testing.T) {
		store, svc := newWithdrawalTestEnv(t)
		paper := seedPaperForWithdrawal(t, store, constants.PaperStatusSubmitted, 1)
		if _, err := svc.Apply(ctx, 2, paper.ID, dto.CreateWithdrawalRequest{Reason: "非本人尝试撤稿：越权操作"}); err == nil {
			t.Fatalf("expected permission denied for non owner")
		}
	})
}

func TestWithdrawalServiceProcess(t *testing.T) {
	ctx := context.Background()

	t.Run("approve withdraws paper and closes reviews", func(t *testing.T) {
		store, svc := newWithdrawalTestEnv(t)
		paper := seedPaperForWithdrawal(t, store, constants.PaperStatusExternalReview, 1)
		review := &model.Review{PaperID: paper.ID, ReviewerID: 9, Status: constants.ReviewStatusAccepted}
		if err := store.reviews.Create(ctx, review); err != nil {
			t.Fatalf("create review: %v", err)
		}
		done := &model.Review{PaperID: paper.ID, ReviewerID: 10, Status: constants.ReviewStatusCompleted}
		if err := store.reviews.Create(ctx, done); err != nil {
			t.Fatalf("create completed review: %v", err)
		}
		w, err := svc.Apply(ctx, 1, paper.ID, dto.CreateWithdrawalRequest{
			Reason: "作者主动撤稿：另投他刊", AltHandlingNote: "审稿意见无需继续处理",
		})
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		processed, err := svc.Process(ctx, 2, w.ID, dto.ProcessWithdrawalRequest{Approve: true, ProcessResult: "同意撤稿"})
		if err != nil {
			t.Fatalf("process approve: %v", err)
		}
		if processed.Status != constants.WithdrawalStatusApproved {
			t.Errorf("expected approved, got %s", processed.Status)
		}
		if processed.ProcessResult != "同意撤稿" || processed.ProcessedAt == nil {
			t.Errorf("expected process result and processed_at recorded, got %+v", processed)
		}
		p, _ := store.papers.FindByID(ctx, paper.ID)
		if p.Status != constants.PaperStatusWithdrawn {
			t.Errorf("expected paper withdrawn, got %s", p.Status)
		}
		unfinished, _ := store.reviews.FindByID(ctx, review.ID)
		if unfinished.Status != constants.ReviewStatusClosed {
			t.Errorf("expected unfinished review closed, got %s", unfinished.Status)
		}
		completed, _ := store.reviews.FindByID(ctx, done.ID)
		if completed.Status != constants.ReviewStatusCompleted {
			t.Errorf("expected completed review unchanged, got %s", completed.Status)
		}
	})

	t.Run("reject restores original flow", func(t *testing.T) {
		store, svc := newWithdrawalTestEnv(t)
		paper := seedPaperForWithdrawal(t, store, constants.PaperStatusRevision, 1)
		w, err := svc.Apply(ctx, 1, paper.ID, dto.CreateWithdrawalRequest{
			Reason: "作者主动撤稿：修改困难", AltHandlingNote: "修稿中止说明",
		})
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		if _, err := svc.Process(ctx, 2, w.ID, dto.ProcessWithdrawalRequest{Approve: false, ProcessResult: "驳回：建议继续修改"}); err != nil {
			t.Fatalf("process reject: %v", err)
		}
		p, _ := store.papers.FindByID(ctx, paper.ID)
		if p.Status != constants.PaperStatusRevision {
			t.Errorf("expected paper status restored to revision, got %s", p.Status)
		}
		// 驳回后原流程恢复：修稿可继续推进
		plagiarismSvc := NewPlagiarismService(store, newTestConfig(), newTestLogger())
		paperSvc := NewPaperService(store, plagiarismSvc, newTestConfig(), newTestLogger())
		if _, err := paperSvc.Revise(ctx, 1, paper.ID, dto.ReviseRequest{
			FileKey: "papers/v2.pdf", FileName: "v2.pdf", ResponseLetter: "已逐条回复审稿意见并完成修改。",
		}); err != nil {
			t.Fatalf("revise should be allowed after rejection: %v", err)
		}
	})

	t.Run("double process rejected", func(t *testing.T) {
		store, svc := newWithdrawalTestEnv(t)
		paper := seedPaperForWithdrawal(t, store, constants.PaperStatusSubmitted, 1)
		w, err := svc.Apply(ctx, 1, paper.ID, dto.CreateWithdrawalRequest{Reason: "作者主动撤稿：重复处理校验"})
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		if _, err := svc.Process(ctx, 2, w.ID, dto.ProcessWithdrawalRequest{Approve: true, ProcessResult: "同意"}); err != nil {
			t.Fatalf("first process: %v", err)
		}
		if _, err := svc.Process(ctx, 2, w.ID, dto.ProcessWithdrawalRequest{Approve: true, ProcessResult: "再次处理"}); err == nil {
			t.Fatalf("expected error for double process")
		}
	})
}

func TestWithdrawalFreezeGuards(t *testing.T) {
	ctx := context.Background()
	store, svc := newWithdrawalTestEnv(t)
	paper := seedPaperForWithdrawal(t, store, constants.PaperStatusRevision, 1)
	reviewer := &model.User{Username: "frozen-rev", Password: "x", RealName: "审稿人", Role: constants.RoleReviewer}
	if err := store.users.Create(ctx, reviewer); err != nil {
		t.Fatalf("create reviewer: %v", err)
	}
	review := &model.Review{PaperID: paper.ID, ReviewerID: reviewer.ID, Status: constants.ReviewStatusAccepted}
	if err := store.reviews.Create(ctx, review); err != nil {
		t.Fatalf("create review: %v", err)
	}
	if _, err := svc.Apply(ctx, 1, paper.ID, dto.CreateWithdrawalRequest{
		Reason: "作者主动撤稿：冻结流程校验", AltHandlingNote: "冻结期间不推进",
	}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	plagiarismSvc := NewPlagiarismService(store, newTestConfig(), newTestLogger())
	paperSvc := NewPaperService(store, plagiarismSvc, newTestConfig(), newTestLogger())
	reviewSvc := NewReviewService(store, newTestLogger())

	assertFrozen := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s: expected frozen error", name)
		}
		appErr, ok := err.(*util.AppError)
		if !ok || appErr.Code != constants.ErrWithdrawalPending {
			t.Fatalf("%s: expected ErrWithdrawalPending, got %v", name, err)
		}
	}

	_, err := paperSvc.Revise(ctx, 1, paper.ID, dto.ReviseRequest{
		FileKey: "papers/v2.pdf", FileName: "v2.pdf", ResponseLetter: "冻结期修稿应被拒绝。",
	})
	assertFrozen("revise", err)

	err = reviewSvc.Submit(ctx, review.ID, reviewer.ID, dto.SubmitReviewRequest{
		Decision: constants.ReviewDecisionAccept, Comments: "冻结期提交审稿应被拒绝。",
	})
	assertFrozen("review submit", err)

	err = reviewSvc.Respond(ctx, review.ID, reviewer.ID, true)
	assertFrozen("review respond", err)

	if _, err := reviewSvc.Assign(ctx, paper.ID, reviewer.ID); err == nil {
		t.Fatalf("assign: expected error (duplicate invite or frozen)")
	}

	_, err = plagiarismSvc.Rerun(ctx, paper.ID)
	assertFrozen("plagiarism rerun", err)

	// 冻结期间记录仍可查看
	if _, err := paperSvc.Detail(ctx, paper.ID); err != nil {
		t.Fatalf("detail should be viewable while frozen: %v", err)
	}
	if _, err := reviewSvc.ListByPaper(ctx, paper.ID); err != nil {
		t.Fatalf("reviews should be viewable while frozen: %v", err)
	}
	if _, err := plagiarismSvc.GetByPaper(ctx, paper.ID); err == nil {
		t.Fatalf("expected plagiarism not found (no check record), but not frozen error")
	} else if appErr, ok := err.(*util.AppError); ok && appErr.Code == constants.ErrWithdrawalPending {
		t.Fatalf("plagiarism view should not be frozen, got %v", err)
	}
}
