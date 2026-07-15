new first;
new second;

main()
{
    new localFirst;
    static localSecond;
    new values[] = {1, 2, 3};
    return first + second + localFirst + localSecond + values[0];
}
