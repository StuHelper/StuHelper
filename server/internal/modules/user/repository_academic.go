package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetAcademicStudentByXH 根据学号查询教务系统学生记录
func (r *Repository) GetAcademicStudentByXH(ctx context.Context, xh string) (*AcademicStudent, error) {
	var item AcademicStudent
	err := r.db.QueryRow(ctx, `
		SELECT xh, xm, sfzjlxdm, sfzjh, yxdm, zydm, bjdm,
		       xznj, rxnj, pyccdm, xslbdm, sjh, dzxx,
		       xjztdm, sfzx, sfzj, synced_at
		FROM academic.buaa_students
		WHERE xh = $1
	`, xh).Scan(
		&item.XH, &item.XM, &item.SFZJLXDM, &item.SFZJH, &item.YXDM, &item.ZYDM, &item.BJDM,
		&item.XZNJ, &item.RXNJ, &item.PYCCDM, &item.XSLBDM, &item.SJH, &item.DZXX,
		&item.XJZTDM, &item.SFZX, &item.SFZJ, &item.SyncedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetAcademicStudentByXH: %w", err)
	}
	return &item, nil
}

// FindAcademicStudentsByPersonUID 根据证件号查询教务系统学生记录
func (r *Repository) FindAcademicStudentsByPersonUID(ctx context.Context, sfzjlxdm, sfzjh string) ([]AcademicStudent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT xh, xm, sfzjlxdm, sfzjh, yxdm, zydm, bjdm,
		       xznj, rxnj, pyccdm, xslbdm, sjh, dzxx,
		       xjztdm, sfzx, sfzj, synced_at
		FROM academic.buaa_students
		WHERE sfzjh = $1
	`, sfzjh)
	if err != nil {
		return nil, fmt.Errorf("FindAcademicStudentsByPersonUID: %w", err)
	}
	defer rows.Close()

	list := make([]AcademicStudent, 0, 4)
	for rows.Next() {
		var item AcademicStudent
		if err := rows.Scan(
			&item.XH, &item.XM, &item.SFZJLXDM, &item.SFZJH, &item.YXDM, &item.ZYDM, &item.BJDM,
			&item.XZNJ, &item.RXNJ, &item.PYCCDM, &item.XSLBDM, &item.SJH, &item.DZXX,
			&item.XJZTDM, &item.SFZX, &item.SFZJ, &item.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("FindAcademicStudentsByPersonUID scan: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// GetInternalUserID 根据外部ID获取内部用户ID
func (r *Repository) GetInternalUserID(ctx context.Context, externalID string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE external_id = $1`, externalID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("GetInternalUserID: %w", err)
	}
	return id, nil
}
