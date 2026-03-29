create type image_status as enum ('pending', 'processing', 'completed', 'failed');

create table if not exists images
(
    id            uuid primary key         default gen_random_uuid(),
    filename      text         not null,
    status        image_status not null    default 'pending',
    original_url  text         not null    default '',
    processed_url text         not null    default '',
    created       timestamp with time zone default now(),
    updated       timestamp with time zone default now()
);
