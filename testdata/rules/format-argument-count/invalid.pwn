main()
{
    new output[64];
    printf("%d %s", 1);
    printf("%d", 1, 2);
    format(output, sizeof (output), "%d");
    SendClientMessage(0, -1, "value %d");
}
