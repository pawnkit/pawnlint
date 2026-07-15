#define LOG_BOTH(%0) do { print(%0); printf("Logged: %s", %0); } while (0)

main()
{
    LOG_BOTH("server started");
}
