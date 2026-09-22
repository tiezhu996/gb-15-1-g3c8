package service

import (
	"context"
	"fmt"

	"github.com/paperflow/paperflow/internal/model"
	"github.com/paperflow/paperflow/internal/repository"
)

// ---- fake 仓储 ----

type fakeUserRepo struct {
	repository.UserRepository
	users  map[uint]*model.User
	byName map[string]*model.User
	nextID uint
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[uint]*model.User{}, byName: map[string]*model.User{}}
}

func (f *fakeUserRepo) Create(ctx context.Context, u *model.User) error {
	if _, ok := f.byName[u.Username]; ok {
		return fmt.Errorf("duplicate username %s", u.Username)
	}
	f.nextID++
	u.ID = f.nextID
	f.users[u.ID] = u
	f.byName[u.Username] = u
	return nil
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id uint) (*model.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserRepo) FindByUsername(ctx context.Context, name string) (*model.User, error) {
	if u, ok := f.byName[name]; ok {
		return u, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserRepo) Update(ctx context.Context, u *model.User) error {
	f.users[u.ID] = u
	f.byName[u.Username] = u
	return nil
}

type fakePaperRepo struct {
	repository.PaperRepository
	papers map[uint]*model.Paper
	nextID uint
}

func newFakePaperRepo() *fakePaperRepo {
	return &fakePaperRepo{papers: map[uint]*model.Paper{}}
}

func (f *fakePaperRepo) Create(ctx context.Context, p *model.Paper) error {
	f.nextID++
	p.ID = f.nextID
	f.papers[p.ID] = p
	return nil
}

func (f *fakePaperRepo) Update(ctx context.Context, p *model.Paper) error {
	f.papers[p.ID] = p
	return nil
}

func (f *fakePaperRepo) FindByID(ctx context.Context, id uint) (*model.Paper, error) {
	if p, ok := f.papers[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakePaperRepo) FindByIDForUpdate(ctx context.Context, id uint) (*model.Paper, error) {
	if p, ok := f.papers[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakePaperRepo) FindByIDWithDetail(ctx context.Context, id uint) (*model.Paper, error) {
	if p, ok := f.papers[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakePaperRepo) List(ctx context.Context, filter model.PaperFilter, page, size int) ([]model.Paper, int64, error) {
	var items []model.Paper
	var total int64
	for _, p := range f.papers {
		if filter.Status != "" && p.Status != filter.Status {
			continue
		}
		if filter.SubmitterID > 0 && p.SubmitterID != filter.SubmitterID {
			continue
		}
		items = append(items, *p)
		total++
	}
	return items, total, nil
}

type fakeReviewRepo struct {
	repository.ReviewRepository
	reviews map[uint]*model.Review
	nextID  uint
}

func newFakeReviewRepo() *fakeReviewRepo {
	return &fakeReviewRepo{reviews: map[uint]*model.Review{}}
}

func (f *fakeReviewRepo) Create(ctx context.Context, r *model.Review) error {
	f.nextID++
	r.ID = f.nextID
	f.reviews[r.ID] = r
	return nil
}

func (f *fakeReviewRepo) Update(ctx context.Context, r *model.Review) error {
	f.reviews[r.ID] = r
	return nil
}

func (f *fakeReviewRepo) FindByID(ctx context.Context, id uint) (*model.Review, error) {
	if r, ok := f.reviews[id]; ok {
		return r, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeReviewRepo) FindByIDForUpdate(ctx context.Context, id uint) (*model.Review, error) {
	if r, ok := f.reviews[id]; ok {
		return r, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeReviewRepo) ListByPaper(ctx context.Context, paperID uint) ([]model.Review, error) {
	var items []model.Review
	for _, r := range f.reviews {
		if r.PaperID == paperID {
			items = append(items, *r)
		}
	}
	return items, nil
}

func (f *fakeReviewRepo) FindInviteByPaperReviewer(ctx context.Context, paperID, reviewerID uint) (*model.Review, error) {
	for _, r := range f.reviews {
		if r.PaperID == paperID && r.ReviewerID == reviewerID {
			return r, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeReviewRepo) CloseUnfinishedByPaper(ctx context.Context, paperID uint) (int64, error) {
	var n int64
	for _, r := range f.reviews {
		if r.PaperID == paperID && (r.Status == "invited" || r.Status == "accepted") {
			r.Status = "closed"
			n++
		}
	}
	return n, nil
}

type fakeRevisionRepo struct {
	repository.RevisionRepository
	revisions []model.Revision
	nextID    uint
}

func newFakeRevisionRepo() *fakeRevisionRepo {
	return &fakeRevisionRepo{}
}

func (f *fakeRevisionRepo) Create(ctx context.Context, r *model.Revision) error {
	f.nextID++
	r.ID = f.nextID
	f.revisions = append(f.revisions, *r)
	return nil
}

func (f *fakeRevisionRepo) ListByPaper(ctx context.Context, paperID uint) ([]model.Revision, error) {
	var items []model.Revision
	for _, r := range f.revisions {
		if r.PaperID == paperID {
			items = append(items, r)
		}
	}
	return items, nil
}

type fakePlagiarismRepo struct {
	repository.PlagiarismRepository
	checks map[uint]*model.PlagiarismCheck
	nextID uint
}

func newFakePlagiarismRepo() *fakePlagiarismRepo {
	return &fakePlagiarismRepo{checks: map[uint]*model.PlagiarismCheck{}}
}

func (f *fakePlagiarismRepo) Create(ctx context.Context, c *model.PlagiarismCheck) error {
	f.nextID++
	c.ID = f.nextID
	f.checks[c.ID] = c
	return nil
}

func (f *fakePlagiarismRepo) Update(ctx context.Context, c *model.PlagiarismCheck) error {
	f.checks[c.ID] = c
	return nil
}

func (f *fakePlagiarismRepo) FindByPaper(ctx context.Context, paperID uint) (*model.PlagiarismCheck, error) {
	for _, c := range f.checks {
		if c.PaperID == paperID {
			return c, nil
		}
	}
	return nil, repository.ErrNotFound
}

type fakeAuditRepo struct {
	repository.AuditLogRepository
	logs []model.AuditLog
}

func newFakeAuditRepo() *fakeAuditRepo {
	return &fakeAuditRepo{}
}

func (f *fakeAuditRepo) Create(ctx context.Context, l *model.AuditLog) error {
	f.logs = append(f.logs, *l)
	return nil
}

type fakeWithdrawalRepo struct {
	repository.WithdrawalRepository
	withdrawals map[uint]*model.WithdrawalRequest
	nextID      uint
}

func newFakeWithdrawalRepo() *fakeWithdrawalRepo {
	return &fakeWithdrawalRepo{withdrawals: map[uint]*model.WithdrawalRequest{}}
}

func (f *fakeWithdrawalRepo) Create(ctx context.Context, w *model.WithdrawalRequest) error {
	for _, x := range f.withdrawals {
		if x.PaperID == w.PaperID && x.Status == "pending" && w.Status == "pending" {
			return fmt.Errorf("duplicate pending withdrawal for paper %d", w.PaperID)
		}
	}
	f.nextID++
	w.ID = f.nextID
	f.withdrawals[w.ID] = w
	return nil
}

func (f *fakeWithdrawalRepo) Update(ctx context.Context, w *model.WithdrawalRequest) error {
	f.withdrawals[w.ID] = w
	return nil
}

func (f *fakeWithdrawalRepo) FindByID(ctx context.Context, id uint) (*model.WithdrawalRequest, error) {
	if w, ok := f.withdrawals[id]; ok {
		return w, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeWithdrawalRepo) FindByIDForUpdate(ctx context.Context, id uint) (*model.WithdrawalRequest, error) {
	return f.FindByID(ctx, id)
}

func (f *fakeWithdrawalRepo) FindPendingByPaper(ctx context.Context, paperID uint) (*model.WithdrawalRequest, error) {
	for _, w := range f.withdrawals {
		if w.PaperID == paperID && w.Status == "pending" {
			return w, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeWithdrawalRepo) ExistsPendingByPaper(ctx context.Context, paperID uint) (bool, error) {
	_, err := f.FindPendingByPaper(ctx, paperID)
	return err == nil, nil
}

func (f *fakeWithdrawalRepo) List(ctx context.Context, filter model.WithdrawalFilter, page, size int) ([]model.WithdrawalRequest, int64, error) {
	var items []model.WithdrawalRequest
	for _, w := range f.withdrawals {
		if filter.Status != "" && w.Status != filter.Status {
			continue
		}
		if filter.ApplicantID > 0 && w.ApplicantID != filter.ApplicantID {
			continue
		}
		if filter.PaperID > 0 && w.PaperID != filter.PaperID {
			continue
		}
		items = append(items, *w)
	}
	return items, int64(len(items)), nil
}

func (f *fakeWithdrawalRepo) ListByPaper(ctx context.Context, paperID uint) ([]model.WithdrawalRequest, error) {
	var items []model.WithdrawalRequest
	for _, w := range f.withdrawals {
		if w.PaperID == paperID {
			items = append(items, *w)
		}
	}
	return items, nil
}

type fakeStore struct {
	users       *fakeUserRepo
	papers      *fakePaperRepo
	reviews     *fakeReviewRepo
	revisions   *fakeRevisionRepo
	plagiarism  *fakePlagiarismRepo
	withdrawals *fakeWithdrawalRepo
	audit       *fakeAuditRepo
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		users:       newFakeUserRepo(),
		papers:      newFakePaperRepo(),
		reviews:     newFakeReviewRepo(),
		revisions:   newFakeRevisionRepo(),
		plagiarism:  newFakePlagiarismRepo(),
		withdrawals: newFakeWithdrawalRepo(),
		audit:       newFakeAuditRepo(),
	}
}

func (f *fakeStore) Transaction(ctx context.Context, fn func(repository.Store) error) error {
	return fn(f)
}

func (f *fakeStore) UserRepository() repository.UserRepository     { return f.users }
func (f *fakeStore) PaperRepository() repository.PaperRepository   { return f.papers }
func (f *fakeStore) ReviewRepository() repository.ReviewRepository { return f.reviews }
func (f *fakeStore) RevisionRepository() repository.RevisionRepository {
	return f.revisions
}
func (f *fakeStore) PlagiarismRepository() repository.PlagiarismRepository {
	return f.plagiarism
}
func (f *fakeStore) WithdrawalRepository() repository.WithdrawalRepository {
	return f.withdrawals
}
func (f *fakeStore) AuditLogRepository() repository.AuditLogRepository {
	return f.audit
}
