create table users (
    id uuid primary key,
    username text unique not null,
    created_at timestamp default now()
);

create table messages (
    id uuid primary key,
    sender_id uuid,
    receiver_id uuid,
    body text,
    created_at timestamp default now()
);

alter table messages
add column if not exists chat_id uuid;

create index if not exists messages_chat_id_idx
on messages(chat_id);

create table if not exists chats (
    id uuid primary key,
    created_at timestamp default now()
);

create table if not exists chat_participants (
    chat_id uuid references chats(id),
    user_id uuid references users(id),
    primary key (chat_id, user_id)
);

create index if not exists messages_chat_created_idx
on messages(chat_id, created_at desc);

create index if not exists chat_participants_user_idx
on chat_participants(user_id);

create index if not exists chat_participants_chat_idx
on chat_participants(chat_id);