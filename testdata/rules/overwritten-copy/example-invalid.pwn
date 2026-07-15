CopyName(const first[], const second[])
{
	new name[16];
	memcpy(name, first, 0, 16 * 4);
	memcpy(name, second, 0, 16 * 4);
    return name[0];
}
