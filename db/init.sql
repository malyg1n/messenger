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