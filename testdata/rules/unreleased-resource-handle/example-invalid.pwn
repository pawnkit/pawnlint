main()
{
    new File:log = fopen("server.log");
    fwrite(log, "server started");
}
