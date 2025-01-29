/* Return section compression header.
   Copyright (C) 2015 Red Hat, Inc.
   This file is part of elfutils.

   This file is free software; you can redistribute it and/or modify
   it under the terms of either

     * the GNU Lesser General Public License as published by the Free
       Software Foundation; either version 3 of the License, or (at
       your option) any later version

   or

     * the GNU General Public License as published by the Free
       Software Foundation; either version 2 of the License, or (at
       your option) any later version

   or both in parallel, as here.

   elfutils is distributed in the hope that it will be useful, but
   WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
   General Public License for more details.

   You should have received copies of the GNU General Public License and
   the GNU Lesser General Public License along with this program.  If
   not, see <http://www.gnu.org/licenses/>.  */

#ifdef HAVE_CONFIG_H
# include <config.h>
#endif

#include <libelf.h>
#include "libelfP.h"
#include "common.h"

#ifndef LIBELFBITS
# define LIBELFBITS 32
#endif


ElfW2(LIBELFBITS,Chdr) *
elfw2(LIBELFBITS,getchdr) (Elf_Scn *scn)
{
  ElfW2(LIBELFBITS,Shdr) *shdr = elfw2(LIBELFBITS,getshdr) (scn);
  if (shdr == NULL)
    return NULL;

  /* Must have SHF_COMPRESSED flag set.  Allocated or no bits sections
     can never be compressed.  */
  if ((shdr->sh_flags & SHF_ALLOC) != 0)
    {
      __libelf_seterrno (ELF_E_INVALID_SECTION_FLAGS);
      return NULL;
    }

  if (shdr->sh_type == SHT_NULL
      || shdr->sh_type == SHT_NOBITS)
    {
      __libelf_seterrno (ELF_E_INVALID_SECTION_TYPE);
      return NULL;
    }

  if ((shdr->sh_flags & SHF_COMPRESSED) == 0)
    {
      __libelf_seterrno (ELF_E_NOT_COMPRESSED);
      return NULL;
    }

  /* This makes sure the data is in the correct format, so we don't
     need to swap fields. */
  Elf_Data *d  = elf_getdata (scn, NULL);
  if (d == NULL)
    return NULL;

  if (d->d_size < sizeof (ElfW2(LIBELFBITS,Chdr)) || d->d_buf == NULL)
    {
      __libelf_seterrno (ELF_E_INVALID_DATA);
      return NULL;
    }

  return (ElfW2(LIBELFBITS,Chdr) *) d->d_buf;
}
