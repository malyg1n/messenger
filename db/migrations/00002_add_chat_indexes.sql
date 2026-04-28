-- +goose Up
create index if not exists chat_participants_user_idx
on chat_participants(user_id);

create index if not exists chat_participants_chat_idx
on chat_participants(chat_id);

-- +goose Down
drop index if exists chat_participants_chat_idx;
drop index if exists chat_participants_user_idx;
