package main

import (
	"context"
	"database/sql"
	"time"
)

var (
	ErrNoRows = sql.ErrNoRows
)

func QueueAdd(args string) (id int64, e error) {
	now := time.Now().Unix()
	res, e := DB.Exec("insert into `queue` (`args`, `status`, `tm_added`, `output`) VALUES(?, ?, ?, '')", args, -1, now)
	if e != nil {
		return
	}

	id, e = res.LastInsertId()
	return
}

func QueueUpdate(ctx context.Context, id int64, status int, output []byte) (e error) {
	now := time.Now().Unix()
	res, e := DB.ExecContext(ctx, "update `queue` set `status` = ?, `output` = ?, `tm_finished` = ? WHERE id = ? limit 1", status, output, now, id)
	if e != nil {
		return
	}
	_, e = res.RowsAffected()
	return
}

func QueueStatus(id string) (state int, output string, e error) {
	e = DB.QueryRow("SELECT `status`, `output` FROM queue WHERE id = ?", id).Scan(&state, &output)
	return
}
