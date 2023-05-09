drop table if exists queue;
create table queue(
	id     integer not null primary key autoincrement,
	sendto text    not null,
	url    text    not null,
	title  text    not null,
	links  text    not null,
	status text    not null default ''
);

-- name: Push :exec
insert into queue(sendto, url) values (?, ?);

-- name: Pop :one
select * from queue where status = '';

-- name: SetFailed :exec
update queue set status = ?;

-- name: SetSuccess :exec
update queue set
	title =?,
	links =?,
	status='success';
