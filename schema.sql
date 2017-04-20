create database `notes`;

create table `notes` (
  `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `note` VARCHAR(1024) NOT NULL,
  `create_date` TIMESTAMP NOT NULL,
  `expiration_date` TIMESTAMP
);

create table `tags` (
  `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `key` VARCHAR(256) NOT NULL,
  `value` VARCHAR(256) NOT NULL,
  `create_date` TIMESTAMP NOT NULL
);
