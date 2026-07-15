main()
{
    new File:log = fopen("first.log");
    fclose(log);
    log = fopen("second.log");
    fclose(log);
}
