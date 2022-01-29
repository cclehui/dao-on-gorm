## 测试表

```
drop table if exists `cclehui_test_a`;
CREATE TABLE `cclehui_test_a` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `version` int(10) unsigned NOT NULL DEFAULT '99' COMMENT 'test' ,
  `weight` decimal(10,2) unsigned NOT NULL DEFAULT '0.00' COMMENT 'test',
  `age` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT 'test',
  `extra` varchar(255) NOT NULL DEFAULT '' COMMENT 'test',
  `created_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='测试表';

drop table if exists `cclehui_test_b`;
CREATE TABLE `cclehui_test_b` (
  `column_id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'column_id',
  `version` int(10) unsigned NOT NULL DEFAULT '99' COMMENT 'test',
  `weight` decimal(10,2) unsigned NOT NULL DEFAULT '0.00' COMMENT 'test',
  `age` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT 'test',
  `extra` varchar(255) NOT NULL DEFAULT '' COMMENT 'test',
  `created_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '更新时间',
  PRIMARY KEY (`column_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4  COMMENT='测试表：非ID主键';

drop table if exists `cclehui_test_c`;
CREATE TABLE `cclehui_test_c` (
  `column_id` int(10) unsigned NOT NULL COMMENT 'column_id',
  `version` int(10) unsigned NOT NULL DEFAULT '99' COMMENT 'test',
  `weight` decimal(10,2) unsigned NOT NULL DEFAULT '0.00' COMMENT 'test',
  `age` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT 'test',
  `extra` varchar(255) NOT NULL DEFAULT '' COMMENT 'test',
  `created_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '更新时间',
  PRIMARY KEY (`column_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='测试表：非自增主键';

drop table if exists `cclehui_test_d`;
CREATE TABLE `cclehui_test_d` (
  `user_id` int(10) unsigned NOT NULL COMMENT 'user_id',
  `column_id` int(10) unsigned NOT NULL COMMENT 'column_id',
  `version` int(10) unsigned NOT NULL DEFAULT '99' COMMENT 'test',
  `weight` decimal(10,2) unsigned NOT NULL DEFAULT '0.00' COMMENT 'test',
  `age` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT 'test',
  `extra` varchar(255) NOT NULL DEFAULT '' COMMENT 'test',
  `created_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '更新时间',
  PRIMARY KEY (`user_id`,`column_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='测试表：联合主键';

```
