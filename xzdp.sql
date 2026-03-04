-- xzdp.sql: Database structure for xzdp-go (Heima Dianping clone)
-- Create time: 2026-03-03
-- Note: Import this file in Navicat (xzdp database)

-- ----------------------------
-- Table structure for user (用户表)
-- ----------------------------
DROP TABLE IF EXISTS `user`;
CREATE TABLE `user` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `email` varchar(20) NOT NULL COMMENT '邮箱', -- 适配邮箱登录，修改注释
  `password` varchar(100) DEFAULT NULL COMMENT '密码（加密存储，邮箱登录可空）',
  `nickname` varchar(50) DEFAULT 'xzdp用户' COMMENT '昵称',
  `avatar` varchar(255) DEFAULT 'https://img-blog.csdnimg.cn/20240101000000.png' COMMENT '头像',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_email` (`email`) COMMENT '手机号/邮箱唯一索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- ----------------------------
-- Table structure for email_code (邮箱验证码表)
-- ----------------------------
DROP TABLE IF EXISTS `email_code`;
CREATE TABLE `email_code` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `email` varchar(50) NOT NULL COMMENT '邮箱（原手机号字段，复用）', -- 仅改注释，字段名不变
  `code` varchar(6) NOT NULL COMMENT '6位验证码',
  `expire_time` datetime NOT NULL COMMENT '过期时间（默认5分钟）',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  INDEX `idx_email` (`email`) COMMENT '邮箱索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邮箱验证码表';

-- 可选：插入测试数据
INSERT INTO `user` (`email`, `nickname`) VALUES ('test@qq.com', '测试用户');