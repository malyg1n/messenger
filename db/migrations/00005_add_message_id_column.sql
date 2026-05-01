-- +goose Up

alter table messages
add column if not exists client_message_id uuid;

create unique index if not exists messages_client_id_uindex
on messages(client_message_id);

-- +goose Down

drop index if exists messages_client_id_uindex;

alter table messages
drop column if exists client_message_id;