/* apk_defines.c - Alpine Package Keeper (APK)
 *
 * Copyright (C) 2005-2008 Natanael Copa <n@tanael.org>
 * Copyright (C) 2008-2011 Timo Teräs <timo.teras@iki.fi>
 * All rights reserved.
 *
 * SPDX-License-Identifier: GPL-2.0-only
 */

#include <string.h>

#define ARRAY_SIZE(x)	(sizeof(x) / sizeof((x)[0]))

struct apk_blob {
	long len;
	char *ptr;
};
typedef struct apk_blob apk_blob_t;

static inline apk_blob_t APK_BLOB_STR(const char *str)
{
	return ((apk_blob_t){strlen(str), (void *)(str)});
}

int apk_version_validate(apk_blob_t ver);
int apk_version_compare_blob(apk_blob_t a, apk_blob_t b);
int apk_version_compare(const char *str1, const char *str2);
