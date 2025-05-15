/*

 Target Server Type    : MariaDB
 Target Server Version : 100126
 File Encoding         : 65001
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for queue
-- ----------------------------
DROP TABLE IF EXISTS `queue`;
CREATE TABLE `queue` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `args` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'PENDING|DONE|ERROR',
  `output` longtext COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Stdout+stderr',
  `tm_added` int(10) unsigned NOT NULL COMMENT 'Unixtimestamp added',
  `tm_finished` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=351 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='BGWorker status (for persistence)';

SET FOREIGN_KEY_CHECKS = 1;
