CREATE TABLE "attachments" (
  "id" BIGSERIAL PRIMARY KEY,
  "user_id" bigint NOT NULL,
  "bucket_name" varchar(32) not null default '',
  "hash_id" varchar(255) not null default '',
  "size" bigint default 0,
  "mime_type" varchar(32) default '',
  "pathname" varchar(255) default '',
  "filename" varchar(255) default '',
  "status" int default 0,
  "original_mime_type" varchar(64),

  "checksum" varchar(128) default '',
  "checksum_method" varchar(32) default '',

  "created_at" timestamptz not null default now(),
  "updated_at" timestamptz not null default now()
);

CREATE INDEX idx_attachment_hash ON "attachments" USING BTREE("hash_id");
CREATE INDEX idx_attachment_status ON "attachments" USING BTREE("status");
CREATE INDEX idx_attachment_checksum ON "attachments" USING BTREE("checksum_method", "checksum");