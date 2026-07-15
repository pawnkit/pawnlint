Check(const first[], const second[])
{
    new full[16];
    new ranged[16];
    memcpy(full, first, 0, 16 * 4);
    memcpy(full, second, 0, 16 * 4);
    Consume(full);
    memcpy(ranged[2], first, 0, 8);
    memcpy(ranged, second, 0, 16 * 4);
    Consume(ranged);
}
