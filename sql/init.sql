CREATE TABLE IF NOT EXISTS auth_user (
    id bigint generated always as identity primary key,
    username varchar(128) not null unique,
    email varchar(255) not null unique,
    role varchar(128) not null,
    password varchar(255) not null,
    created_at timestamptz not null default current_timestamp,
    last_login timestamptz not null default current_timestamp,
    attr jsonb not null default '{}'::jsonb
);
CREATE TABLE IF NOT EXISTS auth_event (
    id bigint generated always as identity primary key,
    action varchar(128) not null,
    detail jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default current_timestamp
);
CREATE TABLE IF NOT EXISTS auth_strike_record (
    id bigint generated always as identity primary key,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    reason TEXT COMMENT '违规原因',
    evidence TEXT COMMENT '证据（URL、截图路径等）',
    severity TINYINT DEFAULT 1 COMMENT '严重程度 1-3',
    strike_weight TINYINT DEFAULT 1 COMMENT '本次扣分权重（默认1分）',
    operator_id BIGINT COMMENT '操作员ID（系统操作可为空）',
    source VARCHAR(50) COMMENT '来源：system/admin/user_report/api',
    ip_address VARCHAR(45) COMMENT '用户IP',
    user_agent TEXT COMMENT '用户代理',
    metadata JSON COMMENT '扩展元数据',
    status VARCHAR(20) DEFAULT 'active' COMMENT '状态：active/revoked/expired',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at),
    INDEX idx_type_status (violation_type, status)
) COMMENT '用户违规记录表';
