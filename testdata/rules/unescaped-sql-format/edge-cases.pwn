main()
{
    new query[128];
    new name[MAX_PLAYER_NAME];
    new city[32];
    mysql_format(1, query, sizeof(query), "SELECT * FROM users WHERE name = '%e' AND city = '%s'", name, city);
    mysql_format(1, query, sizeof(query), "SELECT * FROM users WHERE name = '%s'", "literal-name");
}
