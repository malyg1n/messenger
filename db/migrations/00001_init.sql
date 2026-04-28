-- +goose Up
create table if not exists users (
    id uuid primary key,
    username text unique not null,
    created_at timestamp default now()
);

create table if not exists messages (
    id uuid primary key,
    sender_id uuid,
    receiver_id uuid,
    body text,
    created_at timestamp default now()
);

create table if not exists chats (
    id uuid primary key,
    created_at timestamp default now()
);

create table if not exists chat_participants (
    chat_id uuid references chats(id),
    user_id uuid references users(id),
    primary key (chat_id, user_id)
);

-- +goose Down
drop table if exists chat_participants;
drop table if exists chats;
drop table if exists messages;
drop table if exists users;
