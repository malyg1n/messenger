-- +goose Up

alter table chats add column if not exists chat_key text;

update chats c
set chat_key = sub.chat_key
from (
    select
        cp.chat_id,
        string_agg(cp.user_id::text, '_' order by cp.user_id::text) as chat_key,
        count(*) as participants_count
    from chat_participants cp
    group by cp.chat_id
) sub
where c.id = sub.chat_id
and sub.participants_count = 2
and c.chat_key is null;

create unique index if not exists chats_chat_key_uindex
on chats(chat_key)
where chat_key is not null;

-- +goose Down

drop index if exists chats_chat_key_uindex;
alter table chats drop column if exists chat_key;