create table chat (
    id bigint primary key,
    type int not null,
    message_status int not null default 0,
    timezone varchar(50) not null default ''
);

create table verses_cron (
    chat_id bigint not null references chat(id),
    cron varchar(30),
    unique(chat_id, cron)
);

create table random_time_verses (
    id int primary key,
    chat_id bigint not null references chat(id),
    weekday int not null default -1,
    start_time int not null,
    duration int not null
);

create table next_sends (
    random_time_id int not null references random_time_verses(id) on delete cascade,
    timestamp timestamptz not null,
    unique(random_time_id, timestamp)
);

create table stats (
    date date not null,
    name varchar(30) not null,
    count int not null,
    unique(date, name)
);

create table stats_list_chats (
    date date not null,
    name varchar(30) not null,
    chat_id bigint not null,
    unique(date, name, chat_id)
);

create table keys (
    name varchar(30) primary key,
    min_key int not null
);

insert into keys(name, min_key) values ('random_time_verses', 1);
