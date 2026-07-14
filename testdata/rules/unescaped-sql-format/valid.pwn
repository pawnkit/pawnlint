main()
{
    new query[128];
    new name[MAX_PLAYER_NAME];
    mysql_format(1, query, sizeof(query), "SELECT * FROM users WHERE name = '%e'", name);
    mysql_format(1, query, sizeof(query), "SELECT * FROM `%s` WHERE id = %d", "users", 1);
    mysql_format(1, query, sizeof(query), "SELECT * FROM users WHERE id = %d", 1);
}
