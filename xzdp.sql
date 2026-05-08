CREATE DATABASE IF NOT EXISTS xzdp;
USE xzdp;

-- ----------------------------
-- Table structure for user
-- ----------------------------
DROP TABLE IF EXISTS `user`;
CREATE TABLE `user` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `email` varchar(20) NOT NULL COMMENT '邮箱',
  `password` varchar(100) DEFAULT NULL COMMENT '密码',
  `nickname` varchar(50) DEFAULT 'xzdp用户' COMMENT '昵称',
  `avatar` varchar(255) DEFAULT 'https://img-blog.csdnimg.cn/20240101000000.png' COMMENT '头像',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- ----------------------------
-- Table structure for shop_type
-- ----------------------------
DROP TABLE IF EXISTS `shop_type`;
CREATE TABLE `shop_type` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '类型ID',
  `name` varchar(50) NOT NULL COMMENT '类型名称',
  `icon` varchar(255) DEFAULT NULL COMMENT '图标',
  `sort` int DEFAULT 0,
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户类型表';

INSERT INTO `shop_type` (`name`,`icon`,`sort`) 
VALUES 
("美食","https://img.example.com/food.png",10),
("酒店","https://img.example.com/hotel.png",9),
("休闲娱乐","https://img.example.com/entertain.png",8);

-- ----------------------------
-- Table structure for shop
-- ----------------------------
DROP TABLE IF EXISTS `shop`;
CREATE TABLE `shop` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '商户ID',
  `name` varchar(100) NOT NULL COMMENT '商户名称',
  `type_id` bigint NOT NULL COMMENT '类型ID',
  `images` varchar(1000) DEFAULT NULL,
  `area` varchar(20) DEFAULT NULL,
  `address` varchar(255) NOT NULL,
  `x` decimal(10,6) DEFAULT NULL,
  `y` decimal(10,6) DEFAULT NULL,
  `avg_score` decimal(2,1) DEFAULT '5.0',
  `sold` int DEFAULT '0',
  `comments` int DEFAULT '0',
  `price_range` varchar(20) DEFAULT NULL,
  `open_hours` varchar(50) DEFAULT NULL,
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_type_id` (`type_id`),
  INDEX `idx_location` (`x`,`y`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户表';

INSERT INTO `shop` (`name`,`type_id`,`address`,`avg_score`,`x`,`y`)
VALUES
("海底捞火锅(望京店)",1,"北京市朝阳区望京街9号",4.9,116.472147,39.992881),
("西贝莜面村(国贸店)",1,"北京市朝阳区建国门外大街1号",4.8,116.467321,39.908851),
("全季酒店(中关村店)",2,"北京市海淀区中关村大街",4.7,116.321456,39.987621),
("如家酒店(北京站店)",2,"北京市东城区北京站街",4.5,116.441211,39.905566),
("万达影城(通州店)",3,"北京市通州区万达广场",4.8,116.641211,39.897766),
("乐酷KTV(朝阳路店)",3,"北京市朝阳区朝阳路",4.6,116.511211,39.912345);

-- ----------------------------
-- Table structure for voucher 优惠券主表
-- ----------------------------
DROP TABLE IF EXISTS `voucher`;
CREATE TABLE `voucher` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键',
  `shop_id` bigint NOT NULL COMMENT '商铺id',
  `title` varchar(255) NOT NULL COMMENT '标题',
  `sub_title` varchar(255) DEFAULT NULL COMMENT '副标题',
  `rules` varchar(1024) DEFAULT NULL COMMENT '使用规则',
  `pay_value` bigint NOT NULL COMMENT '支付金额',
  `actual_value` bigint NOT NULL COMMENT '抵扣金额',
  `type` tinyint NOT NULL DEFAULT '0' COMMENT '0普通 1秒杀',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '1上架 2下架',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_shop_id` (`shop_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='优惠券主表';

-- ----------------------------
-- 🔥 修复重点：seckill_vouchers 带自增 id
-- ----------------------------
DROP TABLE IF EXISTS `seckill_voucher`;
CREATE TABLE `seckill_voucher` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `voucher_id` bigint NOT NULL COMMENT '关联优惠券ID',
  `stock` int NOT NULL COMMENT '库存',
  `begin_time` timestamp NOT NULL COMMENT '开始时间',
  `end_time` timestamp NOT NULL COMMENT '结束时间',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_voucher_id` (`voucher_id`),
  INDEX `idx_time` (`begin_time`,`end_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='秒杀优惠券表';

-- ----------------------------
-- 秒杀订单表
-- ----------------------------
DROP TABLE IF EXISTS `seckill_order`;
CREATE TABLE `seckill_order` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '订单ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `voucher_id` bigint NOT NULL COMMENT '优惠券ID',
  `shop_id` bigint NOT NULL COMMENT '商铺ID',
  `status` tinyint(4) NOT NULL DEFAULT '0' COMMENT '订单状态（0-待支付，1-已支付，2-已取消）',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_voucher` (`user_id`,`voucher_id`)COMMENT '一人一单唯一索引',
  INDEX `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='秒杀订单表';

-- ----------------------------
-- 插入优惠券 + 秒杀券数据
-- ----------------------------
INSERT INTO `voucher` (`shop_id`,`title`,`sub_title`,`pay_value`,`actual_value`,`type`,`status`)
VALUES
(1,"海底捞100元代金券","满200可用",100,100,0,1),
(2,"西贝50元代金券","满150可用",50,50,0,1),
(3,"全季酒店8折券","全场通用",80,100,1,1),
(4,"如家酒店50元直减券","无门槛",50,50,1,1);

INSERT INTO `seckill_voucher` (`voucher_id`,`stock`,`begin_time`,`end_time`)
VALUES
(3,100, NOW() - INTERVAL 1 HOUR, NOW() + INTERVAL 7 DAY),
(4,200, NOW() - INTERVAL 1 HOUR, NOW() + INTERVAL 7 DAY);