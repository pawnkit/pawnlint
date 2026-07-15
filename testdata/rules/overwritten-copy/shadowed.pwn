memcpy(dest[], const source[], index, numbytes)
{
    return numbytes;
}

Check(const first[], const second[])
{
    new buffer[16];
    memcpy(buffer, first, 0, 16 * 4);
    memcpy(buffer, second, 0, 16 * 4);
}
