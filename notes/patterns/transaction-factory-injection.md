# Transaction Factory Injection

## What

Services that need atomic multi-repo operations store **factory functions** as struct fields. These factories accept a `database.DBTX` and return a domain repository. Inside `db.WithTx`, the service calls the factory with the transaction to get a tx-scoped repo instance.

## Why

- No global state — factories are injected via the constructor like any other dependency
- The same repository interface works with both pool and transaction (DBTX abstraction)
- Test code can inject mock factories that return in-memory repos
- The service doesn't know or care whether it's talking to pgxpool.Pool or pgx.Tx

## Example

```go
type Service struct {
    orgs          domain.OrgRepository              // pool-backed for reads
    members       domain.MemberRepository
    db            *database.DB
    newOrgRepo    func(database.DBTX) domain.OrgRepository    // factory
    newMemberRepo func(database.DBTX) domain.MemberRepository // factory
    pub           events.Publisher
    log           logger.Logger
}

func New(
    orgs domain.OrgRepository,
    members domain.MemberRepository,
    db *database.DB,
    newOrgRepo func(database.DBTX) domain.OrgRepository,
    newMemberRepo func(database.DBTX) domain.MemberRepository,
    pub events.Publisher,
    log logger.Logger,
) *Service {
    return &Service{
        orgs: orgs, members: members, db: db,
        newOrgRepo: newOrgRepo, newMemberRepo: newMemberRepo,
        pub: pub, log: log,
    }
}
```

Inside a transactional method:

```go
func (s *Service) CreateOrg(ctx context.Context, name, slug string) (domain.Organization, error) {
    const op = "organization: create org"

    org, _ := domain.NewOrganization(name, slug, callerID)
    owner, _ := domain.NewOrgMember(org.ID(), callerID, domain.RoleOwner)

    if err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
        txOrgs := s.newOrgRepo(tx)
        txMembers := s.newMemberRepo(tx)
        if err := txOrgs.Create(ctx, org); err != nil {
            return err
        }
        return txMembers.Add(ctx, owner)
    }); err != nil {
        return domain.Organization{}, canopyerr.Wrap(err, op)
    }

    return org, nil
}
```

At composition root:

```go
orgSvc := organization.New(
    orgRepo, memberRepo, teamRepo, userReader, db,
    func(dbtx database.DBTX) domain.OrgRepository { return postgres.NewOrgRepository(dbtx) },
    func(dbtx database.DBTX) domain.MemberRepository { return postgres.NewMemberRepository(dbtx) },
    pub, log,
)
```

## Contexts Using Transactions

| Context | Operation | What's Atomic |
|---|---|---|
| organization | CreateOrg | org + owner member |
| workspace | CreateWorkspace | workspace + owner member |
| exploration | StartBranch | branch + first leaf |

## Gotchas

- The factory's parameter type is `database.DBTX`, not `pgx.Tx` — this keeps the service decoupled from pgx specifics
- Pool-backed repos (for non-tx reads) and tx-scoped repos (inside WithTx) use the same constructor — `NewXxxRepository(db database.DBTX)`
- First attempt used package-level `var` factories with a `SetTxFactories()` function — this violated "no global state" and was refactored to struct fields
- Don't pass the pool-backed repo into WithTx — queries on a pool connection inside a transaction won't participate in the transaction

## Related

- [[sqlc-mapper-pattern]] — the repos created by these factories use sqlc internally
- [[consumer-side-ports]] — cross-context reads are also injected, but as interfaces not factories
