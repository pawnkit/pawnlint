BuildScratch(limit, const piece[])
{
    for (new i; i < limit; i++) {
        new output[256] = "";
        strcat(output, piece, sizeof output);
        Consume(output);
    }
}

BuildReset(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        output[0] = EOS;
        strcat(output, piece, sizeof output);
    }
    Consume(output);
}

BuildConsumed(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        strcat(output, piece, sizeof output);
        Consume(output);
    }
}

BuildConditional(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        if (i > 0) {
            strcat(output, piece, sizeof output);
        }
    }
    Consume(output);
}

BuildControlled(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        if (i == 2) continue;
        strcat(output, piece, sizeof output);
    }
    Consume(output);
}

BuildSelf(limit)
{
    new output[256] = "x";
    for (new i; i < limit; i++) {
        strcat(output, output, sizeof output);
    }
    Consume(output);
}

BuildUnused(limit, const piece[])
{
    new output[256] = "";
    for (new i; i < limit; i++) {
        strcat(output, piece, sizeof output);
    }
}
