package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/mail"
	"sort"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	batchUserDefaultBalance     = 9999
	batchUserDefaultConcurrency = 3
)

type BatchUserFieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type BatchUserRowError struct {
	RowNo int `json:"row_no"`
	BatchUserFieldError
}

type BatchUserPreviewItem struct {
	RowNo       int                   `json:"row_no"`
	SourceName  string                `json:"source_name"`
	Email       string                `json:"email"`
	Password    string                `json:"password"`
	Username    string                `json:"username"`
	Notes       string                `json:"notes"`
	Balance     float64               `json:"balance"`
	Concurrency int                   `json:"concurrency"`
	Errors      []BatchUserFieldError `json:"errors,omitempty"`
}

type BatchCreateUserInput struct {
	RowNo       int     `json:"row_no"`
	SourceName  string  `json:"source_name"`
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	Username    string  `json:"username"`
	Notes       string  `json:"notes"`
	Balance     float64 `json:"balance"`
	Concurrency int     `json:"concurrency"`
}

type BatchCreatedUserSummary struct {
	RowNo    int    `json:"row_no"`
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type BatchCreateUsersResult struct {
	CreatedCount int                       `json:"created_count"`
	FailedCount  int                       `json:"failed_count"`
	Users        []BatchCreatedUserSummary `json:"users,omitempty"`
	Errors       []BatchUserRowError       `json:"errors,omitempty"`
}

func (s *adminServiceImpl) PreviewBatchUsers(ctx context.Context, names []string) ([]BatchUserPreviewItem, error) {
	cleanNames := compactBatchNames(names)
	if len(cleanNames) == 0 {
		return []BatchUserPreviewItem{}, nil
	}

	baseCount := make(map[string]int, len(cleanNames))
	items := make([]BatchUserPreviewItem, 0, len(cleanNames))
	inputs := make([]BatchCreateUserInput, 0, len(cleanNames))

	for idx, name := range cleanNames {
		username := ""
		if pinyin, err := convertNameToPinyin(name); err == nil && pinyin != "" {
			baseCount[pinyin]++
			username = pinyin
			if count := baseCount[pinyin]; count > 1 {
				username = fmt.Sprintf("%s%d", pinyin, count)
			}
		}

		email := ""
		if username != "" {
			email = username + "@xssio.com"
		}

		password, err := randomHexString(16)
		if err != nil {
			return nil, infraerrors.InternalServer("BATCH_USER_PASSWORD_GEN_FAILED", "failed to generate batch user password").WithCause(err)
		}

		rowNo := idx + 1
		item := BatchUserPreviewItem{
			RowNo:       rowNo,
			SourceName:  name,
			Email:       email,
			Password:    password,
			Username:    username,
			Notes:       "",
			Balance:     batchUserDefaultBalance,
			Concurrency: batchUserDefaultConcurrency,
		}
		items = append(items, item)
		inputs = append(inputs, BatchCreateUserInput{
			RowNo:       rowNo,
			SourceName:  name,
			Email:       email,
			Password:    password,
			Username:    username,
			Notes:       "",
			Balance:     batchUserDefaultBalance,
			Concurrency: batchUserDefaultConcurrency,
		})
	}

	_, errorsByRow, err := s.validateBatchCreateInputs(ctx, inputs)
	if err != nil {
		return nil, err
	}

	for idx := range items {
		if errs := errorsByRow[items[idx].RowNo]; len(errs) > 0 {
			items[idx].Errors = append(items[idx].Errors, errs...)
		}
	}

	return items, nil
}

func (s *adminServiceImpl) CreateUsersBatch(ctx context.Context, input []BatchCreateUserInput) (*BatchCreateUsersResult, error) {
	if len(input) == 0 {
		return &BatchCreateUsersResult{}, nil
	}

	normalized, errorsByRow, err := s.validateBatchCreateInputs(ctx, input)
	if err != nil {
		return nil, err
	}
	if len(errorsByRow) > 0 {
		return &BatchCreateUsersResult{
			CreatedCount: 0,
			FailedCount:  len(input),
			Errors:       flattenBatchErrors(errorsByRow),
		}, nil
	}

	result := &BatchCreateUsersResult{
		Users: make([]BatchCreatedUserSummary, 0, len(normalized)),
	}

	createInContext := func(runCtx context.Context) error {
		for _, item := range normalized {
			user := &User{
				Email:       item.Email,
				Username:    item.Username,
				Notes:       item.Notes,
				Role:        RoleUser,
				Balance:     item.Balance,
				Concurrency: item.Concurrency,
				Status:      StatusActive,
			}
			if err := user.SetPassword(item.Password); err != nil {
				return infraerrors.InternalServer("BATCH_USER_PASSWORD_HASH_FAILED", "failed to hash batch user password").WithCause(err)
			}
			if err := s.userRepo.Create(runCtx, user); err != nil {
				if errors.Is(err, ErrEmailExists) {
					appendBatchError(errorsByRow, item.RowNo, "email", "EMAIL_EXISTS", "email already exists")
					return nil
				}
				return err
			}
			result.Users = append(result.Users, BatchCreatedUserSummary{
				RowNo:    item.RowNo,
				ID:       user.ID,
				Email:    user.Email,
				Username: user.Username,
			})
		}
		return nil
	}

	if s.entClient != nil {
		tx, err := s.entClient.Tx(ctx)
		if err != nil {
			return nil, infraerrors.ServiceUnavailable("BATCH_USER_TX_START_FAILED", "failed to start batch user transaction").WithCause(err)
		}
		defer func() { _ = tx.Rollback() }()

		txCtx := dbent.NewTxContext(ctx, tx)
		if err := createInContext(txCtx); err != nil {
			return nil, err
		}
		if len(errorsByRow) > 0 {
			return &BatchCreateUsersResult{
				CreatedCount: 0,
				FailedCount:  len(input),
				Users:        nil,
				Errors:       flattenBatchErrors(errorsByRow),
			}, nil
		}
		if err := tx.Commit(); err != nil {
			return nil, infraerrors.InternalServer("BATCH_USER_TX_COMMIT_FAILED", "failed to commit batch user transaction").WithCause(err)
		}
	} else {
		if err := createInContext(ctx); err != nil {
			return nil, err
		}
		if len(errorsByRow) > 0 {
			return &BatchCreateUsersResult{
				CreatedCount: 0,
				FailedCount:  len(input),
				Users:        nil,
				Errors:       flattenBatchErrors(errorsByRow),
			}, nil
		}
	}

	for _, created := range result.Users {
		s.assignDefaultSubscriptions(ctx, created.ID)
	}

	result.CreatedCount = len(result.Users)
	result.FailedCount = 0
	return result, nil
}

func compactBatchNames(names []string) []string {
	result := make([]string, 0, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(strings.TrimPrefix(name, "\uFEFF"))
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func (s *adminServiceImpl) validateBatchCreateInputs(ctx context.Context, input []BatchCreateUserInput) ([]BatchCreateUserInput, map[int][]BatchUserFieldError, error) {
	normalized := make([]BatchCreateUserInput, len(input))
	errorsByRow := make(map[int][]BatchUserFieldError)

	emailRows := make(map[string]int, len(input))
	usernameRows := make(map[string]int, len(input))
	uniqueEmails := make(map[string]int, len(input))

	for idx, item := range input {
		rowNo := item.RowNo
		if rowNo <= 0 {
			rowNo = idx + 1
		}

		normalizedItem := BatchCreateUserInput{
			RowNo:       rowNo,
			SourceName:  strings.TrimSpace(item.SourceName),
			Email:       strings.ToLower(strings.TrimSpace(item.Email)),
			Password:    item.Password,
			Username:    strings.TrimSpace(item.Username),
			Notes:       strings.TrimSpace(item.Notes),
			Balance:     item.Balance,
			Concurrency: item.Concurrency,
		}
		normalized[idx] = normalizedItem

		if normalizedItem.Username == "" {
			appendBatchError(errorsByRow, rowNo, "username", "USERNAME_REQUIRED", "username is required")
		} else if len([]rune(normalizedItem.Username)) > 100 {
			appendBatchError(errorsByRow, rowNo, "username", "USERNAME_TOO_LONG", "username must be 100 characters or fewer")
		}

		if normalizedItem.Email == "" {
			appendBatchError(errorsByRow, rowNo, "email", "EMAIL_REQUIRED", "email is required")
		} else if !isValidBatchEmail(normalizedItem.Email) {
			appendBatchError(errorsByRow, rowNo, "email", "INVALID_EMAIL", "email format is invalid")
		}

		if len(normalizedItem.Password) < 6 {
			appendBatchError(errorsByRow, rowNo, "password", "PASSWORD_TOO_SHORT", "password must be at least 6 characters")
		}

		if math.IsNaN(normalizedItem.Balance) || math.IsInf(normalizedItem.Balance, 0) {
			appendBatchError(errorsByRow, rowNo, "balance", "INVALID_BALANCE", "balance must be a valid number")
		} else if normalizedItem.Balance < 0 {
			appendBatchError(errorsByRow, rowNo, "balance", "NEGATIVE_BALANCE", "balance cannot be negative")
		}

		if normalizedItem.Concurrency < 1 {
			appendBatchError(errorsByRow, rowNo, "concurrency", "INVALID_CONCURRENCY", "concurrency must be at least 1")
		}

		if normalizedItem.Email != "" {
			if prevRow, ok := emailRows[normalizedItem.Email]; ok {
				appendBatchError(errorsByRow, prevRow, "email", "DUPLICATE_EMAIL", "email is duplicated in this batch")
				appendBatchError(errorsByRow, rowNo, "email", "DUPLICATE_EMAIL", "email is duplicated in this batch")
			} else {
				emailRows[normalizedItem.Email] = rowNo
				uniqueEmails[normalizedItem.Email] = rowNo
			}
		}

		if normalizedItem.Username != "" {
			if prevRow, ok := usernameRows[normalizedItem.Username]; ok {
				appendBatchError(errorsByRow, prevRow, "username", "DUPLICATE_USERNAME", "username is duplicated in this batch")
				appendBatchError(errorsByRow, rowNo, "username", "DUPLICATE_USERNAME", "username is duplicated in this batch")
			} else {
				usernameRows[normalizedItem.Username] = rowNo
			}
		}
	}

	for email, rowNo := range uniqueEmails {
		exists, err := s.userRepo.ExistsByEmail(ctx, email)
		if err != nil {
			return nil, nil, infraerrors.InternalServer("BATCH_USER_EMAIL_CHECK_FAILED", "failed to validate existing batch user email").WithCause(err)
		}
		if exists {
			appendBatchError(errorsByRow, rowNo, "email", "EMAIL_EXISTS", "email already exists")
		}
	}

	return normalized, errorsByRow, nil
}

func appendBatchError(errorsByRow map[int][]BatchUserFieldError, rowNo int, field, code, message string) {
	if rowNo <= 0 {
		return
	}
	existing := errorsByRow[rowNo]
	for _, item := range existing {
		if item.Field == field && item.Code == code && item.Message == message {
			return
		}
	}
	errorsByRow[rowNo] = append(existing, BatchUserFieldError{
		Field:   field,
		Code:    code,
		Message: message,
	})
}

func flattenBatchErrors(errorsByRow map[int][]BatchUserFieldError) []BatchUserRowError {
	if len(errorsByRow) == 0 {
		return nil
	}

	rowNos := make([]int, 0, len(errorsByRow))
	for rowNo := range errorsByRow {
		rowNos = append(rowNos, rowNo)
	}
	sort.Ints(rowNos)

	flattened := make([]BatchUserRowError, 0, len(errorsByRow))
	for _, rowNo := range rowNos {
		rowErrors := append([]BatchUserFieldError(nil), errorsByRow[rowNo]...)
		sort.Slice(rowErrors, func(i, j int) bool {
			if rowErrors[i].Field == rowErrors[j].Field {
				return rowErrors[i].Code < rowErrors[j].Code
			}
			return rowErrors[i].Field < rowErrors[j].Field
		})
		for _, item := range rowErrors {
			flattened = append(flattened, BatchUserRowError{
				RowNo:               rowNo,
				BatchUserFieldError: item,
			})
		}
	}
	return flattened
}

func isValidBatchEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr != nil && addr.Address == email
}
