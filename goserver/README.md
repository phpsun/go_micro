## Go micro Server
go_micro_init_build.zip，存放构建脚本

操作步骤：
1、本地安装etcd
2、将构建脚本跟项目统一放到～/www/目录下
3、执行构建脚本，启动实例接口
    ~/www/go_micro_init_build/rpc_base.sh api
    ~/www/go_micro_init_build/rpc_base.sh conf
4、请求http://localhost:8801/api/conf/region-info?id=1，查看实例接口

实例接口对应表结构
CREATE TABLE `conf_regions` (
  `id` int(11) NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `name` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '地区字段名',
  `title` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '地区名',
  `parent_id` int(11) NOT NULL COMMENT '上级地区ID',
  `lat` double NOT NULL COMMENT '纬度',
  `lng` double NOT NULL COMMENT '经度',
  `order` int(11) NOT NULL COMMENT '排序位置',
  `is_show` int(2) NOT NULL DEFAULT '1' COMMENT '1展示，0隐藏',
  `geo_pos` varchar(255) COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT 'latitude,longitude',
  PRIMARY KEY (`id`),
  KEY `title` (`title`)
) ENGINE=InnoDB AUTO_INCREMENT=913 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='配置表-地区';