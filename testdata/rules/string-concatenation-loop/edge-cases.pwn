#define APPEND(%0,%1) strcat(%0, %1, sizeof %0)

strcat(dest[], const source[], maxlength)
{
    return maxlength + dest[0] + source[0];
}

CheckShadowed(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        strcat(output, piece, sizeof output);
    }
    Consume(output);
}

CheckMacro(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        APPEND(output, piece);
    }
    Consume(output);
}
