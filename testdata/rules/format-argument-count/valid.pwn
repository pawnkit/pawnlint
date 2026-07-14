#define C_ORANGE "{FF9900}"

main()
{
    new output[64];
    printf("value %05d %.2f %% %q", 1, 1.0, "text");
    printf("%d" " %s", 1, "text");
    format(output, sizeof (output), "%d", 1);
    printf(""C_ORANGE"value %d", 1);
}
