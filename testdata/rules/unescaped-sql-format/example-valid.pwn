#include <a_mysql>

FormatLookup(connection, const name[])
{
    new query[128];
    mysql_format(connection, query, sizeof query, "SELECT id FROM users WHERE name = '%e'", name);
    return query[0];
}
