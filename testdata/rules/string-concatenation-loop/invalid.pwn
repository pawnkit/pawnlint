BuildString(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        strcat(output, piece, sizeof output);
    }
    Consume(output);
}

BuildDefaultLimit(limit, const piece[])
{
    new output[256] = "";
    do {
        strcat(output, piece);
    } while (--limit > 0);
    Consume(output);
}

BuildDelimited(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        strcat(output, piece, sizeof output);
        strcat(output, ",", sizeof output);
    }
    Consume(output);
}
