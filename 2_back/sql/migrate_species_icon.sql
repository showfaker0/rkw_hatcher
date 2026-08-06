-- 图鉴增加图标字段
ALTER TABLE species
  ADD COLUMN icon_url VARCHAR(512) NULL COMMENT '精灵图标 URL' AFTER name;
