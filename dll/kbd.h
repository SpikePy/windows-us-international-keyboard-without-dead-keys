/***************************************************************************\
* kbd.h - minimal keyboard layout table definitions
*
* A small, self-authored subset of the structures and constants that every
* Windows keyboard layout DLL's KbdLayerDescriptor() export relies on
* (KBDTABLES and friends). Written from scratch against the current, real
* struct layout - verified against Microsoft's own kbdus.c reference sample
* (microsoft/Windows-driver-samples) and the published field ordering of
* windows::Win32::UI::Input::KeyboardAndMouse::KBDTABLES - rather than
* copied from the Windows Driver Kit, so this project has no dependency on
* the WDK or on MSKLC being downloaded/installed anywhere in the pipeline.
\***************************************************************************/

#ifndef _KBD_MINIMAL_H_
#define _KBD_MINIMAL_H_

/*
 * Shift-state bit values (bitmask reported by GetModifierBits()).
 */
#define KBDBASE  0
#define KBDSHIFT 1
#define KBDCTRL  2
#define KBDALT   4

#define SHFT_INVALID 0x0F

typedef struct {
    BYTE Vk;
    BYTE ModBits;
} VK_TO_BIT, *PVK_TO_BIT;

#ifdef _MSC_VER
#pragma warning(disable : 4200)
#endif
typedef struct {
    PVK_TO_BIT pVkToBit;
    WORD       wMaxModBits;
    BYTE       ModNumber[]; /* flexible array member: indexed by raw ModBits */
} MODIFIERS, *PMODIFIERS;
#ifdef _MSC_VER
#pragma warning(default : 4200)
#endif

/*
 * VK_TO_WCHARS<n> - a Virtual Key plus <n> Unicode characters, one per
 * shift state. Attributes bit CAPLOK/SGCAPS/CAPLOKALTGR control how
 * CapsLock affects this key.
 */
#define WCH_NONE 0xF000
#define WCH_DEAD 0xF001

#define CAPLOK      0x01
#define SGCAPS      0x02
#define CAPLOKALTGR 0x04

#define TYPEDEF_VK_TO_WCHARS(n) \
    typedef struct _VK_TO_WCHARS##n { \
        BYTE  VirtualKey; \
        BYTE  Attributes; \
        WCHAR wch[n]; \
    } VK_TO_WCHARS##n, *PVK_TO_WCHARS##n;

TYPEDEF_VK_TO_WCHARS(1)
TYPEDEF_VK_TO_WCHARS(2)
TYPEDEF_VK_TO_WCHARS(3)
TYPEDEF_VK_TO_WCHARS(4)
TYPEDEF_VK_TO_WCHARS(5)
TYPEDEF_VK_TO_WCHARS(6)

typedef struct _VK_TO_WCHAR_TABLE {
    PVK_TO_WCHARS1 pVkToWchars;
    BYTE           nModifications;
    BYTE           cbSize;
} VK_TO_WCHAR_TABLE, *PVK_TO_WCHAR_TABLE;

typedef struct {
    DWORD dwBoth;
    WCHAR wchComposed;
} DEADKEY, *PDEADKEY;

typedef struct _VSC_VK {
    BYTE   Vsc;
    USHORT Vk;
} VSC_VK, *PVSC_VK;

typedef struct {
    BYTE   vsc;
    LPWSTR pwsz;
} VSC_LPWSTR, *PVSC_LPWSTR;

typedef struct _LIGATURE1 {
    BYTE   VirtualKey;
    WORD   ModificationNumber;
    WCHAR  wch[1];
} LIGATURE1, *PLIGATURE1;

/*
 * KBDTABLES - the structure KbdLayerDescriptor() returns. Field order
 * matters: it must exactly match what user32/the OS expects.
 */
typedef struct tagKbdLayer {
    PMODIFIERS         pCharModifiers;
    VK_TO_WCHAR_TABLE *pVkToWcharTable;
    PDEADKEY           pDeadKey;
    VSC_LPWSTR        *pKeyNames;
    VSC_LPWSTR        *pKeyNamesExt;
    LPWSTR            *pKeyNamesDead;
    USHORT            *pusVSCtoVK;
    BYTE               bMaxVSCtoVK;
    PVSC_VK            pVSCtoVK_E0;
    PVSC_VK            pVSCtoVK_E1;
    DWORD              fLocaleFlags;
    BYTE               nLgMax;
    BYTE               cbLgEntry;
    PLIGATURE1         pLigature;
    DWORD              dwType;
    DWORD              dwSubType;
} KBDTABLES, *PKBDTABLES;

/*
 * Key event flags, also used as ausVK[]/aE0.../aE1... entry bits.
 */
#define KBDEXT     (USHORT)0x0100
#define KBDMULTIVK (USHORT)0x0200
#define KBDSPECIAL (USHORT)0x0400
#define KBDNUMPAD  (USHORT)0x0800

#endif /* _KBD_MINIMAL_H_ */
