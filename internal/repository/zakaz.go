package repository

import (
	"context"
	"time"

	"github.com/SoulStalker/chz-api-client/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ZakazRepo struct {
	pool *pgxpool.Pool
}

func NewZakazRepo(pool *pgxpool.Pool) *ZakazRepo {
	return &ZakazRepo{pool: pool}
}

const zakazColumns = `
	z.kod, z.data, z.nomer, z.post_kod, z.post_imya, z.post_inn,
	z.data_postavki, z.naklad_kod, z.summa, z.updated_at, z.crpt_doc_id`

func scanZakaz(row interface {
	Scan(dest ...any) error
}) (model.Zakaz, error) {
	var z model.Zakaz
	err := row.Scan(
		&z.Kod, &z.Data, &z.Nomer, &z.PostKod, &z.PostImya, &z.PostINN,
		&z.DataPostavki, &z.NakladKod, &z.Summa, &z.UpdatedAt, &z.CRPTDocID,
	)
	return z, err
}

func (r *ZakazRepo) List(ctx context.Context, dateFrom, dateTo time.Time, offset, limit int) ([]model.Zakaz, int, error) {
	const q = `
		SELECT` + zakazColumns + `
		FROM ord_zakaz z
		WHERE z.data >= $1 AND z.data <= $2
		ORDER BY z.data DESC, z.kod
		LIMIT $3 OFFSET $4`

	rows, err := r.pool.Query(ctx, q, dateFrom, dateTo, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []model.Zakaz
	for rows.Next() {
		z, err := scanZakaz(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, z)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	const countQ = `SELECT COUNT(*) FROM ord_zakaz WHERE data >= $1 AND data <= $2`
	if err := r.pool.QueryRow(ctx, countQ, dateFrom, dateTo).Scan(&total); err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *ZakazRepo) Get(ctx context.Context, kod string) (*model.Zakaz, []model.ZakazLine, error) {
	const q = `SELECT` + zakazColumns + ` FROM ord_zakaz z WHERE z.kod = $1`
	z, err := scanZakaz(r.pool.QueryRow(ctx, q, kod))
	if err != nil {
		return nil, nil, err
	}

	const lq = `
		SELECT id, zakaz_kod, tovar_kod, shk, naim, kolvo, summa
		FROM ord_zakaz_lines
		WHERE zakaz_kod = $1
		ORDER BY id`

	rows, err := r.pool.Query(ctx, lq, kod)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var lines []model.ZakazLine
	for rows.Next() {
		var l model.ZakazLine
		if err := rows.Scan(&l.ID, &l.ZakazKod, &l.TovarKod, &l.SHK, &l.Naim, &l.Kolvo, &l.Summa); err != nil {
			return nil, nil, err
		}
		lines = append(lines, l)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return &z, lines, nil
}

func (r *ZakazRepo) FindByCRPTDoc(ctx context.Context, crptDocID string) (*model.Zakaz, error) {
	const q = `SELECT` + zakazColumns + ` FROM ord_zakaz z WHERE z.crpt_doc_id = $1 LIMIT 1`
	z, err := scanZakaz(r.pool.QueryRow(ctx, q, crptDocID))
	if err != nil {
		return nil, err
	}
	return &z, nil
}

func (r *ZakazRepo) FindByDocIDs(ctx context.Context, ids []string) (map[string]model.Zakaz, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	const q = `SELECT` + zakazColumns + ` FROM ord_zakaz z WHERE z.crpt_doc_id = ANY($1)`
	rows, err := r.pool.Query(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]model.Zakaz)
	for rows.Next() {
		z, err := scanZakaz(rows)
		if err != nil {
			return nil, err
		}
		if z.CRPTDocID != nil {
			result[*z.CRPTDocID] = z
		}
	}
	return result, rows.Err()
}

func (r *ZakazRepo) SetCRPTDocID(ctx context.Context, kod, crptDocID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ord_zakaz SET crpt_doc_id = $1 WHERE kod = $2`,
		crptDocID, kod,
	)
	return err
}

func (r *ZakazRepo) ClearCRPTDocID(ctx context.Context, kod string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ord_zakaz SET crpt_doc_id = NULL WHERE kod = $1`,
		kod,
	)
	return err
}
