new matrix[2][2];

main()
{
    matrix[0][3] = 1;

#if defined FEATURE
    matrix[3] = 1;
#endif
}
