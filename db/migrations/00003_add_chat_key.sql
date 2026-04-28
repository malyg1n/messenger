-- +goose Up
alter table messages
add column if not exists chat_id uuid;

create index if not exists messages_chat_id_idx
on messages(chat_id);

create index if not exists messages_chat_created_idx
on messages(chat_id, created_at desc);

-- +goose Down
drop index if exists messages_chat_created_idx;
drop index if exists messages_chat_id_idx;
alter table messages drop column if exists chat_id;
