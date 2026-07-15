#define LOG_BOTH(%0) print(%0); printf("Logged: %s", %0)

main()
{
    LOG_BOTH("server started");
}
