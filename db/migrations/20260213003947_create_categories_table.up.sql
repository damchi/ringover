CREATE TABLE categories (
    id   BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL UNIQUE
) ENGINE=InnoDB;

INSERT INTO categories (name) VALUES
  ('Backend'), ('Frontend'), ('Bug'), ('Feature');
