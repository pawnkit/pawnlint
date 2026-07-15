Check(const first[], const second[], size)
{
    new read[16];
    new partial[16];
    new dynamic[16];
    new self[16];
    new branched[16];
    new controlled[16];
    memcpy(read, first, 0, 16 * 4);
    Consume(read);
    memcpy(read, second, 0, 16 * 4);
    memcpy(partial, first, 0, 16 * 4);
    memcpy(partial, second, 0, 8 * 4);
    memcpy(dynamic, first, 0, size);
    memcpy(dynamic, second, 0, 16 * 4);
    memcpy(self, first, 0, 16 * 4);
    memcpy(self, self, 0, 16 * 4);
    memcpy(branched, first, 0, 16 * 4);
    if (size > 0) {
        memcpy(branched, second, 0, 16 * 4);
    }
    memcpy(controlled, first, 0, 16 * 4);
    if (size > 0) {
        return;
    }
    memcpy(controlled, second, 0, 16 * 4);
}
