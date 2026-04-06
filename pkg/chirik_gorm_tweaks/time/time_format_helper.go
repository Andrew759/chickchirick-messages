package time

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// TimestampWithTimeZoneMicro - кастомный данных для gorm. Хранит только микросекунды (как БД, в частности
// PostgreSQL.) Отсекает наносекунды
type TimestampWithTimeZoneMicro struct {
	time.Time
}

func (t *TimestampWithTimeZoneMicro) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.Time.Format("2006-01-02T15:04:05.000000Z07:00") + `"`), nil
}

func (t *TimestampWithTimeZoneMicro) Scan(value interface{}) error {
	if val, ok := value.(time.Time); ok {
		t.Time = val.Truncate(time.Microsecond)
		return nil
	}
	return fmt.Errorf("cannot convert %v to TimestampWithTimeZoneMicro", value)
}

func (t TimestampWithTimeZoneMicro) Value() (driver.Value, error) {
	return t.Time.Truncate(time.Microsecond), nil
}
