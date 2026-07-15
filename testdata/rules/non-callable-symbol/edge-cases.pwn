#define DOUBLE(%0) ((%0) * 2)

main()
{
    new value = DOUBLE(21);
    printf("%d", value);
}

forward Handler(playerid);
public Handler(playerid)
{
    return playerid;
}
