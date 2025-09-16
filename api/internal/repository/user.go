package repository

import (
	"auth/.gen/auth/public/model"
	. "auth/.gen/auth/public/table"
	"database/sql"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
)

const (
	RoleAdmin      string = "admin"
	RoleTrusted    string = "trusted"
	RoleMember     string = "member"
	RoleRestricted string = "restricted"
	RoleBanned     string = "banned"
)

type User = model.AuthUser

type UserFilter struct {
	Username      string
	Role          string
	CreatedBefore time.Time
	CreatedAfter  time.Time
}

type UserRepository interface {
	List(filter UserFilter, pageNumber, pageSize int64) ([]*User, error)
	FindByUsername(username string) (*User, error)
	FindByEmail(email string) (*User, error)
	Save(user *User) error
	UpdateLastLogin(user *User) error
	UpdateHashedPassword(user *User) error
	UpdateRole(user *User) error
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) List(filter UserFilter, pageNumber int64, pageSize int64) ([]*User, error) {
	stmt := SELECT(AuthUser.AllColumns).
		FROM(AuthUser)

	if filter.Username != "" {
		stmt = stmt.WHERE(AuthUser.Username.LIKE(String(filter.Username)))
	}
	if filter.Role != "" {
		stmt = stmt.WHERE(AuthUser.Role.EQ(String(filter.Role)))
	}
	if !filter.CreatedBefore.IsZero() {
		stmt = stmt.WHERE(AuthUser.CreatedAt.LT(TimestampzT(filter.CreatedBefore)))
	}
	if !filter.CreatedAfter.IsZero() {
		stmt = stmt.WHERE(AuthUser.CreatedAt.GT(TimestampzT(filter.CreatedAfter)))
	}

	stmt = stmt.
		ORDER_BY(AuthUser.ID.ASC()).
		LIMIT(pageSize).
		OFFSET(pageNumber * pageSize)

	var dest []*User
	err := stmt.Query(r.db, &dest)
	if err == qrm.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return dest, nil
}

func (r *userRepository) FindByUsername(username string) (*User, error) {
	stmt := SELECT(AuthUser.AllColumns).
		FROM(AuthUser).
		WHERE(AuthUser.Username.EQ(String(username)))

	var dest User
	err := stmt.Query(r.db, &dest)
	if err == qrm.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &dest, nil
}

func (r *userRepository) FindByEmail(email string) (*User, error) {
	stmt := SELECT(AuthUser.AllColumns).
		FROM(AuthUser).
		WHERE(AuthUser.Email.EQ(String(email)))

	var dest User
	err := stmt.Query(r.db, &dest)
	if err == qrm.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &dest, nil
}

func (r *userRepository) Save(user *User) error {
	stmt := AuthUser.INSERT(AuthUser.MutableColumns).
		MODEL(user)

	_, err := stmt.Exec(r.db)
	return err
}

func (r *userRepository) UpdateLastLogin(user *User) error {
	stmt := AuthUser.UPDATE(AuthUser.LastLogin).
		SET(TimestampzT(time.Now())).
		WHERE(AuthUser.ID.EQ(Int(user.ID)))

	_, err := stmt.Exec(r.db)
	return err
}

func (r *userRepository) UpdateHashedPassword(user *User) error {
	stmt := AuthUser.UPDATE(AuthUser.Password).
		SET(String(user.Password)).
		WHERE(AuthUser.ID.EQ(Int(user.ID)))

	_, err := stmt.Exec(r.db)
	return err
}

func (r *userRepository) UpdateRole(user *User) error {
	stmt := AuthUser.UPDATE(AuthUser.Role).
		SET(String(user.Role)).
		WHERE(AuthUser.ID.EQ(Int(user.ID)))

	_, err := stmt.Exec(r.db)
	return err
}
