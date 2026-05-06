package apperr

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// FromGORM — конвертирует gorm ошибки в доменные
func FromGORM(err error, entity string) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return NotFound(entity)
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return AlreadyExists(entity)
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return InvalidArgument(fmt.Sprintf("%s has invalid reference", entity))
	case errors.Is(err, gorm.ErrInvalidData):
		return InvalidArgument(fmt.Sprintf("%s has invalid data", entity))
	default:
		// mapping to INTERNAL
		return fmt.Errorf("db: %w", err)
	}
}
