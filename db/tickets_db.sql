-- Создаем таблицу статусов
CREATE TABLE tickets_statuses (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

-- Заполняем таблицу статусов начальными данными
INSERT INTO tickets_statuses (id, name) VALUES 
(1, 'Новый'),
(2, 'В процессе'),
(3, 'Завершен');

-- Добавляем столбец для ID пользователя (целое число)
ALTER TABLE tickets ADD COLUMN user_id INTEGER;

-- Добавляем столбец для имени компьютера (текст)
ALTER TABLE tickets ADD COLUMN computer_name TEXT;