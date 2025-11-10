package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"service_users/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// UserRepository интерфейс для работы с пользователями
type UserRepository interface {
	Create(user *models.User) error
	GetByID(id uuid.UUID) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Update(user *models.User) error
	List(req *models.ListUsersRequest) (*models.ListUsersResponse, error)
	EmailExists(email string) (bool, error)
}

// userRepository реализация UserRepository
type userRepository struct {
	db *sql.DB
}

// NewUserRepository создает новый экземпляр UserRepository
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

// Create создает нового пользователя
func (r *userRepository) Create(user *models.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, roles, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err := r.db.Exec(query,
		user.ID,
		user.Email,
		user.Password,
		user.Name,
		pq.Array(user.Roles),
		user.CreatedAt,
		user.UpdatedAt,
	)
	
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("пользователь с email %s уже существует", user.Email)
		}
		return fmt.Errorf("ошибка создания пользователя: %v", err)
	}

	return nil
}

// GetByID получает пользователя по ID
func (r *userRepository) GetByID(id uuid.UUID) (*models.User, error) {
    query := `
        SELECT id, email, password_hash, name, roles, created_at, updated_at
        FROM users
        WHERE id = $1
    `

    user := &models.User{}
    err := r.db.QueryRow(query, id).Scan(
        &user.ID,
        &user.Email,
        &user.Password,
        &user.Name,
        &user.Roles,
        &user.CreatedAt,
        &user.UpdatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("пользователь с ID %s не найден", id)
        }
        return nil, fmt.Errorf("ошибка получения пользователя: %v", err)
    }
    return user, nil
}

// EmailExists проверяет существование email
func (r *userRepository) EmailExists(email string) (bool, error) {
    query := "SELECT EXISTS(SELECT 1 FROM users WHERE lower(email) = lower($1))"

    var exists bool
    if err := r.db.QueryRow(query, strings.ToLower(email)).Scan(&exists); err != nil {
        return false, fmt.Errorf("ошибка проверки существования email: %v", err)
    }
    return exists, nil
}

// GetByEmail получает пользователя по email
func (r *userRepository) GetByEmail(email string) (*models.User, error) {
    query := `
        SELECT id, email, password_hash, name, roles, created_at, updated_at
        FROM users
        WHERE lower(email) = $1
    `

    user := &models.User{}
    err := r.db.QueryRow(query, strings.ToLower(email)).Scan(
        &user.ID,
        &user.Email,
        &user.Password,
        &user.Name,
        &user.Roles,
        &user.CreatedAt,
        &user.UpdatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("пользователь с email %s не найден", email)
        }
        return nil, fmt.Errorf("ошибка получения пользователя: %v", err)
    }
    return user, nil
}

// Update обновляет данные пользователя
func (r *userRepository) Update(user *models.User) error {
    query := `
        UPDATE users
        SET email = $2, name = $3, roles = $4, updated_at = NOW()
        WHERE id = $1
    `

    result, err := r.db.Exec(query,
        user.ID,
        user.Email,
        user.Name,
        pq.Array(user.Roles),
    )
    if err != nil {
        if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
            return fmt.Errorf("пользователь с email %s уже существует", user.Email)
        }
        return fmt.Errorf("ошибка обновления пользователя: %v", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("ошибка получения количества обновленных строк: %v", err)
    }
    if rowsAffected == 0 {
        return fmt.Errorf("пользователь с ID %s не найден", user.ID)
    }
    return nil
}

// List получает список пользователей с фильтрацией и пагинацией
func (r *userRepository) List(req *models.ListUsersRequest) (*models.ListUsersResponse, error) {
    // Построение WHERE условий
    var conditions []string
    var args []interface{}
    argIndex := 1

    if req.Email != "" {
        conditions = append(conditions, fmt.Sprintf("email ILIKE $%d", argIndex))
        args = append(args, "%"+req.Email+"%")
        argIndex++
    }

    if req.Name != "" {
        conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argIndex))
        args = append(args, "%"+req.Name+"%")
        argIndex++
    }

    if req.Role != "" {
        conditions = append(conditions, fmt.Sprintf("$%d = ANY(roles)", argIndex))
        args = append(args, req.Role)
        argIndex++
    }

    whereClause := ""
    if len(conditions) > 0 {
        whereClause = "WHERE " + strings.Join(conditions, " AND ")
    }

    // Получение общего количества
    countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
    var total int
    if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
        return nil, fmt.Errorf("ошибка подсчета пользователей: %v", err)
    }

    // Получение списка пользователей
    query := fmt.Sprintf(`
        SELECT id, email, password_hash, name, roles, created_at, updated_at
        FROM users
        %s
        ORDER BY created_at DESC
        LIMIT $%d OFFSET $%d
    `, whereClause, argIndex, argIndex+1)

    args = append(args, req.Limit, req.Offset)

    rows, err := r.db.Query(query, args...)
    if err != nil {
        return nil, fmt.Errorf("ошибка получения списка пользователей: %v", err)
    }
    defer rows.Close()

    var users []models.User
    for rows.Next() {
        var user models.User
        if err := rows.Scan(
            &user.ID,
            &user.Email,
            &user.Password,
            &user.Name,
            &user.Roles,
            &user.CreatedAt,
            &user.UpdatedAt,
        ); err != nil {
            return nil, fmt.Errorf("ошибка сканирования пользователя: %v", err)
        }
        // очищаем пароль в выдаче списка
        user.Password = ""
        users = append(users, user)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("ошибка итерации по строкам: %v", err)
    }

    return &models.ListUsersResponse{
        Users:  users,
        Total:  total,
        Limit:  req.Limit,
        Offset: req.Offset,
    }, nil
}
