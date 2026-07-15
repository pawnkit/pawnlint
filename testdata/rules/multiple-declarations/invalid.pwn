new first, second, third;

main()
{
    new localFirst, localSecond;
    for (new row = 0, column = 0; row < 3; row++, column++)
    {
        localFirst += row + column;
    }
    return first + second + third + localFirst + localSecond;
}
