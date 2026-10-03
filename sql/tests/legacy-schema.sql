create table auth_user (
    id bigint generated always as identity primary key,
    username varchar(128) not null unique,
    email varchar(255) not null unique,
    role varchar(128) not null,
    password varchar(255) not null,
    created_at timestamptz not null default current_timestamp,
    last_login timestamptz not null default current_timestamp,
    attr jsonb not null default '{}'::jsonb
);
create table auth_strike_record (
    id bigint generated always as identity primary key,
    user_id bigint not null,
    operator_id bigint,
    reason text not null,
    evidence text not null,
    point smallint not null default 1,
    status smallint not null default 0,
    created_at timestamptz not null default current_timestamp,
    attr jsonb not null default '{}'::jsonb
);
insert into auth_user (username, email, role, password, attr)
values ('legacy-user', 'legacy@example.invalid', 'member', 'preserved-password-hash', '{"legacy":true}');
insert into auth_strike_record (user_id, reason, evidence, status)
values (1, 'existing penalty', 'retained evidence', 0);
