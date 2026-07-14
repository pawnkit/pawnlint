main()
{
    new query[128];
    new name[MAX_PLAYER_NAME];
    mysql_format(1, query, sizeof(query), "SELECT * FROM users WHERE name = '%s'", name);
}
