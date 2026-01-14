package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Rafhael-Viana/m/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// helpers
func parseDateOnly(v string) (time.Time, error) {
	// formato: YYYY-MM-DD
	return time.Parse("2006-01-02", v)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// GET /reports/points
func ReportPoints(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		// filtros comuns
		userID := strings.TrimSpace(q.Get("user_id"))
		deptID := strings.TrimSpace(q.Get("setor_id"))   // opcional
		status := strings.TrimSpace(q.Get("status"))     // open|close
		location := strings.TrimSpace(q.Get("location")) // busca simples (ILIKE)

		// datas (recomendado sempre usar range)
		var (
			from *time.Time
			to   *time.Time
		)
		if v := q.Get("from"); v != "" {
			d, err := parseDateOnly(v)
			if err != nil {
				http.Error(w, "invalid from (use YYYY-MM-DD)", http.StatusBadRequest)
				return
			}
			from = &d
		}
		if v := q.Get("to"); v != "" {
			d, err := parseDateOnly(v)
			if err != nil {
				http.Error(w, "invalid to (use YYYY-MM-DD)", http.StatusBadRequest)
				return
			}
			// boa prática: incluir o dia inteiro => toExclusive = to + 1 day
			d = d.AddDate(0, 0, 1)
			to = &d
		}

		limit := 50
		offset := 0
		if v := q.Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		if v := q.Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}

		where := []string{"1=1"}
		args := []any{}
		argN := 1

		// OBS: aqui uso clock_in como referência de período
		if from != nil {
			where = append(where, fmt.Sprintf("p.clock_in >= $%d", argN))
			args = append(args, *from)
			argN++
		}
		if to != nil {
			where = append(where, fmt.Sprintf("p.clock_in < $%d", argN))
			args = append(args, *to)
			argN++
		}
		if userID != "" {
			where = append(where, fmt.Sprintf("p.user_id = $%d", argN))
			args = append(args, userID)
			argN++
		}
		if status != "" {
			if status != "open" && status != "close" {
				http.Error(w, "invalid status (open|close)", http.StatusBadRequest)
				return
			}
			where = append(where, fmt.Sprintf("p.status = $%d", argN))
			args = append(args, status)
			argN++
		}
		if location != "" {
			// procura em location_in/location_out
			where = append(where, fmt.Sprintf("(p.location_in ILIKE $%d OR p.location_out ILIKE $%d)", argN, argN))
			args = append(args, "%"+location+"%")
			argN++
		}

		// se você tiver users/departments, dá pra filtrar por dept aqui
		joinDept := ""
		if deptID != "" {
			joinDept = "JOIN users u ON u.id = p.user_id"
			where = append(where, fmt.Sprintf("u.setor_id = $%d", argN))
			args = append(args, deptID)
			argN++
		}

		// paginação
		args = append(args, limit, offset)
		limitPos := argN
		offsetPos := argN + 1

		query := fmt.Sprintf(`
			SELECT
				p.id, p.user_id,
				p.clock_in, p.clock_out,
				p.status,
				p.location_in, p.location_out,
				p.photo_in, p.photo_out,
				p.created_at, p.updated_at
			FROM points p
			%s
			WHERE %s
			ORDER BY p.clock_in DESC NULLS LAST, p.created_at DESC
			LIMIT $%d OFFSET $%d
		`, joinDept, strings.Join(where, " AND "), limitPos, offsetPos)

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		rows, err := database.Pool().Query(ctx, query, args...)
		if err != nil {
			http.Error(w, "error fetching report points", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type PointRow struct {
			ID          int        `json:"id"`
			UserID      string     `json:"user_id"`
			ClockIn     *time.Time `json:"clock_in"`
			ClockOut    *time.Time `json:"clock_out,omitempty"`
			Status      string     `json:"status"`
			LocationIn  *string    `json:"location_in"`
			LocationOut *string    `json:"location_out,omitempty"`
			PhotoIn     *string    `json:"photo_in"`
			PhotoOut    *string    `json:"photo_out,omitempty"`
			CreatedAt   time.Time  `json:"created_at"`
			UpdatedAt   time.Time  `json:"updated_at"`
		}

		out := []PointRow{}
		for rows.Next() {
			var p PointRow
			var locIn, locOut, phIn, phOut pgtype.Text

			if err := rows.Scan(
				&p.ID, &p.UserID,
				&p.ClockIn, &p.ClockOut,
				&p.Status,
				&locIn, &locOut,
				&phIn, &phOut,
				&p.CreatedAt, &p.UpdatedAt,
			); err != nil {
				http.Error(w, "error reading rows", http.StatusInternalServerError)
				fmt.Println(err)
				return
			}

			if locIn.Valid {
				p.LocationIn = &locIn.String
			}
			if locOut.Valid {
				p.LocationOut = &locOut.String
			}
			if phIn.Valid {
				p.PhotoIn = &phIn.String
			}
			if phOut.Valid {
				p.PhotoOut = &phOut.String
			}

			out = append(out, p)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"limit":  limit,
			"offset": offset,
			"items":  out,
		})
	}
}

// GET /reports/frequency?group_by=user|department|day
func ReportFrequency(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		groupBy := strings.TrimSpace(q.Get("group_by"))
		if groupBy == "" {
			groupBy = "user"
		}
		if groupBy != "user" && groupBy != "department" && groupBy != "day" {
			http.Error(w, "invalid group_by (user|department|day)", http.StatusBadRequest)
			return
		}

		var from time.Time
		var to time.Time
		// aqui eu recomendo obrigar as datas
		if q.Get("from") == "" || q.Get("to") == "" {
			http.Error(w, "from and to are required (YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		var err error
		from, err = parseDateOnly(q.Get("from"))
		if err != nil {
			http.Error(w, "invalid from (YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		to, err = parseDateOnly(q.Get("to"))
		if err != nil {
			http.Error(w, "invalid to (YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		to = to.AddDate(0, 0, 1) // exclusivo

		// Métricas úteis:
		// - shifts_total: total de registros no período
		// - shifts_closed: quantos estão status='close'
		// - days_worked: quantos dias distintos (base clock_in::date)
		// - hours_worked: soma de (clock_out - clock_in) somente quando fechado
		//
		// Ajuste se quiser contar apenas closed em tudo.
		var query string
		args := []any{from, to}

		switch groupBy {
		case "user":
			query = `
				SELECT
					p.user_id AS key,
					COUNT(*) AS shifts_total,
					COUNT(*) FILTER (WHERE p.status = 'close') AS shifts_closed,
					COUNT(DISTINCT (p.clock_in::date)) AS days_worked,
					COALESCE(SUM(EXTRACT(EPOCH FROM (p.clock_out - p.clock_in))) FILTER (WHERE p.status='close'), 0) AS seconds_worked
				FROM points p
				WHERE p.clock_in >= $1 AND p.clock_in < $2
				GROUP BY p.user_id
				ORDER BY days_worked DESC, shifts_closed DESC
			`

		case "department":
			// precisa de users.department_id e departments.id/name (ajuste nomes)
			query = `
				SELECT
					COALESCE(s.nome, 'Sem setor') AS key,
					COUNT(*) AS shifts_total,
					COUNT(*) FILTER (WHERE p.status = 'close') AS shifts_closed,
					COUNT(DISTINCT (p.clock_in::date)) AS days_worked,
					COALESCE(SUM(EXTRACT(EPOCH FROM (p.clock_out - p.clock_in))) FILTER (WHERE p.status='close'), 0) AS seconds_worked
				FROM points p
				LEFT JOIN users u ON u.user_id = p.user_id
				LEFT JOIN setores s ON s.setor_id = u.setor_id
				WHERE p.clock_in >= $1 AND p.clock_in < $2
				GROUP BY COALESCE(s.nome, 'Sem setor')
				ORDER BY days_worked DESC, shifts_closed DESC
			`

		case "day":
			query = `
				SELECT
					(p.clock_in::date)::text AS key,
					COUNT(*) AS shifts_total,
					COUNT(*) FILTER (WHERE p.status = 'close') AS shifts_closed,
					COUNT(DISTINCT p.user_id) AS users_present,
					COALESCE(SUM(EXTRACT(EPOCH FROM (p.clock_out - p.clock_in))) FILTER (WHERE p.status='close'), 0) AS seconds_worked
				FROM points p
				WHERE p.clock_in >= $1 AND p.clock_in < $2
				GROUP BY (p.clock_in::date)
				ORDER BY (p.clock_in::date) ASC
			`
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		rows, err := database.Pool().Query(ctx, query, args...)
		if err != nil {
			http.Error(w, "error fetching frequency report", http.StatusInternalServerError)
			fmt.Println(err)
			return
		}
		defer rows.Close()

		type Row struct {
			Key          string  `json:"key"`
			ShiftsTotal  int64   `json:"shifts_total"`
			ShiftsClosed int64   `json:"shifts_closed"`
			DaysWorked   int64   `json:"days_worked"`
			UsersPresent *int64  `json:"users_present,omitempty"`
			HoursWorked  float64 `json:"hours_worked"`
		}

		out := []Row{}
		for rows.Next() {
			var (
				key           string
				shiftsTotal   int64
				shiftsClosed  int64
				daysWorked    int64
				usersPresent  *int64
				secondsWorked float64
			)

			if groupBy == "day" {
				var up int64
				if err := rows.Scan(&key, &shiftsTotal, &shiftsClosed, &up, &secondsWorked); err != nil {
					http.Error(w, "error reading rows", http.StatusInternalServerError)
					fmt.Println(err)
					return
				}
				usersPresent = &up
			} else {
				if err := rows.Scan(&key, &shiftsTotal, &shiftsClosed, &daysWorked, &secondsWorked); err != nil {
					http.Error(w, "error reading rows", http.StatusInternalServerError)
					fmt.Println(err)
					return
				}
			}

			hours := secondsWorked / 3600.0
			out = append(out, Row{
				Key:          key,
				ShiftsTotal:  shiftsTotal,
				ShiftsClosed: shiftsClosed,
				DaysWorked:   daysWorked,
				UsersPresent: usersPresent,
				HoursWorked:  hours,
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"group_by": groupBy,
			"from":     q.Get("from"),
			"to":       q.Get("to"),
			"items":    out,
		})
	}
}
