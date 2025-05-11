WITH latest_timestamps AS (
    SELECT 
        computer_id,
        (SELECT MAX(timestamp) FROM processors WHERE computer_id = c.computer_id) AS last_cpu,
        (SELECT MAX(timestamp) FROM memory WHERE computer_id = c.computer_id) AS last_mem,
        (SELECT MAX(timestamp) FROM disks WHERE computer_id = c.computer_id) AS last_disk,
        (SELECT MAX(timestamp) FROM network_adapters WHERE computer_id = c.computer_id AND is_active = TRUE) AS last_net
    FROM computers c
    WHERE computer_id = ($Computer)
),

network_totals AS (
    SELECT
        computer_id,
        SUM(received_mb) AS total_received_mb,
        SUM(sent_mb) AS total_sent_mb
    FROM network_adapters
    WHERE computer_id = ($Computer)
    GROUP BY computer_id
),

system_data AS (
    SELECT
        1 AS display_order,
        'Система' AS category,
        host_name AS description,
        os_name AS system_name,
        os_version AS system_version,
        NULL AS additional_info,
        NULL::timestamp AS timestamp
    FROM computers
    WHERE computer_id = ($Computer)
),

user_data AS (
    SELECT
        2 AS display_order,
        'Пользователь' AS category,
        user_name AS description,
        NULL AS system_name,
        NULL AS system_version,
        'Группа: ' || COALESCE(gid, 'не указана') || ', Домашняя папка: ' || COALESCE(home_directory, 'не указана') AS additional_info,
        NULL::timestamp AS timestamp
    FROM computers
    WHERE computer_id = ($Computer))
),

uptime_data AS (
    SELECT
        3 AS display_order,
        'Время работы' AS category,
        'Запуск системы' AS description,
        NULL AS system_name,
        NULL AS system_version,
        TO_CHAR(boot_time, 'DD.MM.YYYY HH24:MI:SS') || ', Аптайм: ' || 
        EXTRACT(HOUR FROM AGE(NOW(), boot_time)) || ' ч ' ||
        EXTRACT(MINUTE FROM AGE(NOW(), boot_time)) || ' мин' AS additional_info,
        NULL::timestamp AS timestamp
    FROM computers
    WHERE computer_id = ($Computer))
),

network_usage_data AS (
    SELECT
        4 AS display_order,
        'Сетевая активность' AS category,
        'Всего данных' AS description,
        NULL AS system_name,
        NULL AS system_version,
        ROUND(total_received_mb, 2) || ' MB получено, ' || ROUND(total_sent_mb, 2) || ' MB отправлено' AS additional_info,
        NULL::timestamp AS timestamp
    FROM network_totals
    WHERE computer_id = ($Computer)
),

cpu_data AS (
    SELECT
        5 AS display_order,
        'Процессор' AS category,
        p.model AS description,
        NULL AS system_name,
        NULL AS system_version,
        p.usage_percent || '%, ' || p.core_count || ' ядер, ' || p.thread_count || ' потоков, ' || p.clock_speed || ' GHz' AS additional_info,
        p.timestamp
    FROM processors p, latest_timestamps lt
    WHERE p.computer_id = ($Computer) AND p.timestamp = lt.last_cpu
),

memory_data AS (
    SELECT
        6 AS display_order,
        'Память' AS category,
        total_memory_gb || ' GB' AS description,
        NULL AS system_name,
        NULL AS system_version,
        usage_percent || '%, ' || used_memory_gb || ' GB / ' || free_memory_gb || ' GB' AS additional_info,
        timestamp
    FROM memory m, latest_timestamps lt
    WHERE m.computer_id = ($Computer) AND m.timestamp = lt.last_mem
),

disk_data AS (
    SELECT
        7 AS display_order,
        'Диски' AS category,
        drive_letter AS description,
        NULL AS system_name,
        NULL AS system_version,
        usage_percent || '%, ' || used_space_gb || ' / ' || total_space_gb || ' GB' AS additional_info,
        timestamp
    FROM disks d, latest_timestamps lt
    WHERE d.computer_id = ($Computer) AND d.timestamp = lt.last_disk
),

network_data AS (
    SELECT
        8 AS display_order,
        'Сеть' AS category,
        adapter_name AS description,
        NULL AS system_name,
        NULL AS system_version,
        upload_speed_mbps || '↑ / ' || download_speed_mbps || '↓ Mbps, MAC: ' || mac_address AS additional_info,
        timestamp
    FROM network_adapters na, latest_timestamps lt
    WHERE na.computer_id = ($Computer) AND na.is_active = TRUE AND na.timestamp = lt.last_net
),

update_data AS (
    SELECT
        9 AS display_order,
        'Обновление' AS category,
        'Последние данные' AS description,
        NULL AS system_name,
        NULL AS system_version,
        TO_CHAR(GREATEST(lt.last_cpu, lt.last_mem, lt.last_disk, lt.last_net), 'DD.MM.YYYY HH24:MI:SS') AS additional_info,
        GREATEST(lt.last_cpu, lt.last_mem, lt.last_disk, lt.last_net) AS timestamp
    FROM latest_timestamps lt
),

combined_data AS (
    SELECT * FROM system_data
    UNION ALL SELECT * FROM user_data
    UNION ALL SELECT * FROM uptime_data
    UNION ALL SELECT * FROM network_usage_data
    UNION ALL SELECT * FROM cpu_data
    UNION ALL SELECT * FROM memory_data
    UNION ALL SELECT * FROM disk_data
    UNION ALL SELECT * FROM network_data
    UNION ALL SELECT * FROM update_data
)

SELECT 
    category,
    description,
    system_name,
    system_version,
    additional_info,
    CASE 
        WHEN timestamp IS NULL THEN NULL
        ELSE TO_CHAR(timestamp, 'YYYY-MM-DD HH24:MI:SS')
    END AS timestamp
FROM combined_data
ORDER BY display_order;