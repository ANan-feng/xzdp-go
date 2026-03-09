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

-- ----------------------------
-- Table structure for shop (商户表)
-- ----------------------------
DROP TABLE IF EXISTS `shop`;
CREATE TABLE `shop` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '商户ID',
  `name` varchar(100) NOT NULL COMMENT '商户名称',
  `type_id` bigint NOT NULL COMMENT '商户类型ID',
  `images` varchar(1000) DEFAULT NULL COMMENT '商户图片，多个图片用,分隔',
  `area` varchar(20) DEFAULT NULL COMMENT '商户所在区域',
  `address` varchar(255) NOT NULL COMMENT '商户详细地址',
  `x` decimal(10,6) DEFAULT NULL COMMENT '经度',
  `y` decimal(10,6) DEFAULT NULL COMMENT '纬度',
  `avg_score` decimal(2,1) DEFAULT '5.0' COMMENT '评分',
  `sold` int DEFAULT '0' COMMENT '销量',
  `comments` int DEFAULT '0' COMMENT '评论数',
  `price_range` varchar(20) DEFAULT NULL COMMENT '价格区间',
  `open_hours` varchar(50) DEFAULT NULL COMMENT '营业时间',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `idx_type_id` (`type_id`) COMMENT '商户类型索引',
  INDEX `idx_location` (`x`,`y`) COMMENT '地理位置索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户表';

-- 插入测试数据
INSERT INTO `shop` (`name`, `type_id`, `address`, `avg_score`) 
VALUES ('测试商户1', 1, '北京市朝阳区测试路1号', 4.8),
       ('测试商户2', 2, '上海市浦东新区测试路2号', 4.5);


-- ----------------------------
-- Table structure for shop_type (商户类型表)
-- ----------------------------
DROP TABLE IF EXISTS `shop_type`;
CREATE TABLE `shop_type` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '类型ID',
  `name` varchar(50) NOT NULL COMMENT '类型名称（如美食、酒店、休闲）',
  `icon` varchar(255) DEFAULT NULL COMMENT '类型图标URL',
  `sort` int DEFAULT 0 COMMENT '排序权重（越大越靠前）',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`) COMMENT '类型名称唯一'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户类型表';

-- 插入测试数据
INSERT INTO `shop_type` (`name`, `icon`, `sort`) 
VALUES ('美食', 'https://img.example.com/food.png', 10),
       ('酒店', 'https://img.example.com/hotel.png', 9),
       ('休闲娱乐', 'https://img.example.com/entertain.png', 8);




-- 优惠券主表（对应黑马点评tb_voucher）
DROP TABLE IF EXISTS `voucher`;
CREATE TABLE `voucher` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键',
  `shop_id` bigint NOT NULL COMMENT '商铺id',
  `title` varchar(255) NOT NULL COMMENT '代金券标题',
  `sub_title` varchar(255) DEFAULT NULL COMMENT '副标题',
  `rules` varchar(1024) DEFAULT NULL COMMENT '使用规则',
  `pay_value` bigint NOT NULL COMMENT '支付金额（分）',
  `actual_value` bigint NOT NULL COMMENT '抵扣金额（分）',
  `type` tinyint NOT NULL DEFAULT '0' COMMENT '0-普通券；1-秒杀券',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '1-上架；2-下架；3-过期',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `idx_shop_id` (`shop_id`) COMMENT '商铺ID索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='优惠券主表';
-- ----------------------------
-- 秒杀优惠券表（一对一关联优惠券主表，对应黑马点评tb_seckill_voucher）
DROP TABLE IF EXISTS `seckill_voucher`;
CREATE TABLE `seckill_voucher` (
  `voucher_id` bigint NOT NULL COMMENT '关联优惠券ID',
  `stock` int NOT NULL COMMENT '库存',
  `begin_time` timestamp NOT NULL COMMENT '生效时间',
  `end_time` timestamp NOT NULL COMMENT '失效时间',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`voucher_id`) COMMENT '与优惠券一对一',
  INDEX `idx_time` (`begin_time`,`end_time`) COMMENT '时间范围索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='秒杀优惠券表';


-- 秒杀订单表（保留原逻辑，补充商铺ID可选）
DROP TABLE IF EXISTS `seckill_order`;
CREATE TABLE `seckill_order` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '订单ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `voucher_id` bigint NOT NULL COMMENT '优惠券ID',
  `shop_id` bigint NOT NULL COMMENT '商铺ID',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_voucher` (`user_id`,`voucher_id`) COMMENT '一人一单',
  INDEX `idx_user_id` (`user_id`) COMMENT '用户ID索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='秒杀订单表';