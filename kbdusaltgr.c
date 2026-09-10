/***************************************************************************\
* Module Name: KBDUSALTGR.C
*
* US QWERTY keyboard layout with AltGr accented characters.
*
* Based on Microsoft's reference kbdus.c (United States layout); adds
* AltGr (Ctrl+Alt) and Shift+AltGr characters on U, O, A, S.
\***************************************************************************/

#include <windows.h>
#include "kbd.h"

#pragma data_seg(".data")
#define ALLOC_SECTION_LDATA

/***************************************************************************\
* ausVK[] - Virtual Scan Code to Virtual Key conversion table
\***************************************************************************/

static ALLOC_SECTION_LDATA USHORT ausVK[] = {
    /* 00 */ 0,
    /* 01 */ VK_ESCAPE,
    /* 02 */ '1', /* 03 */ '2', /* 04 */ '3', /* 05 */ '4',
    /* 06 */ '5', /* 07 */ '6', /* 08 */ '7', /* 09 */ '8',
    /* 0A */ '9', /* 0B */ '0',
    /* 0C */ VK_OEM_MINUS, /* 0D */ VK_OEM_PLUS,
    /* 0E */ VK_BACK, /* 0F */ VK_TAB,
    /* 10 */ 'Q', /* 11 */ 'W', /* 12 */ 'E', /* 13 */ 'R',
    /* 14 */ 'T', /* 15 */ 'Y', /* 16 */ 'U', /* 17 */ 'I',
    /* 18 */ 'O', /* 19 */ 'P',
    /* 1A */ VK_OEM_4, /* 1B */ VK_OEM_6,
    /* 1C */ VK_RETURN, /* 1D */ VK_LCONTROL,
    /* 1E */ 'A', /* 1F */ 'S', /* 20 */ 'D', /* 21 */ 'F',
    /* 22 */ 'G', /* 23 */ 'H', /* 24 */ 'J', /* 25 */ 'K', /* 26 */ 'L',
    /* 27 */ VK_OEM_1, /* 28 */ VK_OEM_7, /* 29 */ VK_OEM_3,
    /* 2A */ VK_LSHIFT, /* 2B */ VK_OEM_5,
    /* 2C */ 'Z', /* 2D */ 'X', /* 2E */ 'C', /* 2F */ 'V',
    /* 30 */ 'B', /* 31 */ 'N', /* 32 */ 'M',
    /* 33 */ VK_OEM_COMMA, /* 34 */ VK_OEM_PERIOD, /* 35 */ VK_OEM_2,
    /* 36 */ VK_RSHIFT | KBDEXT,
    /* 37 */ VK_MULTIPLY | KBDMULTIVK,
    /* 38 */ VK_LMENU, /* 39 */ VK_SPACE, /* 3A */ VK_CAPITAL,
    /* 3B */ VK_F1, /* 3C */ VK_F2, /* 3D */ VK_F3, /* 3E */ VK_F4,
    /* 3F */ VK_F5, /* 40 */ VK_F6, /* 41 */ VK_F7, /* 42 */ VK_F8,
    /* 43 */ VK_F9, /* 44 */ VK_F10,
    /* 45 */ VK_NUMLOCK | KBDEXT | KBDMULTIVK,
    /* 46 */ VK_SCROLL | KBDMULTIVK,
    /* 47 */ VK_HOME    | KBDNUMPAD | KBDSPECIAL,
    /* 48 */ VK_UP      | KBDNUMPAD | KBDSPECIAL,
    /* 49 */ VK_PRIOR   | KBDNUMPAD | KBDSPECIAL,
    /* 4A */ VK_SUBTRACT,
    /* 4B */ VK_LEFT    | KBDNUMPAD | KBDSPECIAL,
    /* 4C */ VK_CLEAR   | KBDNUMPAD | KBDSPECIAL,
    /* 4D */ VK_RIGHT   | KBDNUMPAD | KBDSPECIAL,
    /* 4E */ VK_ADD,
    /* 4F */ VK_END     | KBDNUMPAD | KBDSPECIAL,
    /* 50 */ VK_DOWN    | KBDNUMPAD | KBDSPECIAL,
    /* 51 */ VK_NEXT    | KBDNUMPAD | KBDSPECIAL,
    /* 52 */ VK_INSERT  | KBDNUMPAD | KBDSPECIAL,
    /* 53 */ VK_DELETE  | KBDNUMPAD | KBDSPECIAL,
    /* 54 */ VK_SNAPSHOT,
    /* 55 */ 0,
    /* 56 */ VK_OEM_102,
    /* 57 */ VK_F11, /* 58 */ VK_F12,
    /* 59 */ 0, /* 5A */ 0, /* 5B */ 0, /* 5C */ 0, /* 5D */ 0,
    /* 5E */ 0, /* 5F */ 0, /* 60 */ 0, /* 61 */ 0, /* 62 */ 0, /* 63 */ 0,
    /* 64 */ VK_F13, /* 65 */ VK_F14, /* 66 */ VK_F15, /* 67 */ VK_F16,
    /* 68 */ VK_F17, /* 69 */ VK_F18, /* 6A */ VK_F19, /* 6B */ VK_F20,
    /* 6C */ VK_F21, /* 6D */ VK_F22, /* 6E */ VK_F23,
    /* 6F */ 0, /* 70 */ 0, /* 71 */ 0, /* 72 */ 0, /* 73 */ 0,
    /* 74 */ 0, /* 75 */ 0,
    /* 76 */ VK_F24,
    /* 77 */ 0, /* 78 */ 0, /* 79 */ 0, /* 7A */ 0, /* 7B */ 0,
    /* 7C */ 0, /* 7D */ 0, /* 7E */ 0
};

static ALLOC_SECTION_LDATA VSC_VK aE0VscToVk[] = {
    { 0x1D, VK_RCONTROL | KBDEXT },
    { 0x35, VK_DIVIDE   | KBDEXT },
    { 0x37, VK_SNAPSHOT | KBDEXT },
    { 0x38, VK_RMENU    | KBDEXT },
    { 0x47, VK_HOME     | KBDEXT },
    { 0x48, VK_UP       | KBDEXT },
    { 0x49, VK_PRIOR    | KBDEXT },
    { 0x4B, VK_LEFT     | KBDEXT },
    { 0x4D, VK_RIGHT    | KBDEXT },
    { 0x4F, VK_END      | KBDEXT },
    { 0x50, VK_DOWN     | KBDEXT },
    { 0x51, VK_NEXT     | KBDEXT },
    { 0x52, VK_INSERT   | KBDEXT },
    { 0x53, VK_DELETE   | KBDEXT },
    { 0x5B, VK_LWIN     | KBDEXT },
    { 0x5C, VK_RWIN     | KBDEXT },
    { 0x5D, VK_APPS     | KBDEXT },
    { 0x1C, VK_RETURN   | KBDEXT },
    { 0,    0                   }
};

static ALLOC_SECTION_LDATA VSC_VK aE1VscToVk[] = {
    { 0x1D, VK_PAUSE },
    { 0,    0        }
};

/***************************************************************************\
* aVkToBits[] / CharModifiers - three shifter keys (Shift, Ctrl, Alt);
* Alt+Ctrl (AltGr) and Shift+Alt+Ctrl are valid, producing AltGr characters.
\***************************************************************************/
static ALLOC_SECTION_LDATA VK_TO_BIT aVkToBits[] = {
    { VK_SHIFT,   KBDSHIFT },
    { VK_CONTROL, KBDCTRL  },
    { VK_MENU,    KBDALT   },
    { 0,          0        }
};

static ALLOC_SECTION_LDATA MODIFIERS CharModifiers = {
    &aVkToBits[0],
    7,
    {
    //  Modification# //  Keys Pressed
    //  ============= // =============
        0,             // (none)
        1,             // Shift
        2,             // Control
        SHFT_INVALID,  // Shift + Control
        SHFT_INVALID,  // Alt
        SHFT_INVALID,  // Shift + Alt
        3,             // Control + Alt (AltGr)
        4              // Shift + Control + Alt (Shift+AltGr)
    }
};

/***************************************************************************\
* aVkToWch2[] - keys with only base/shift characters (no AltGr mapping)
\***************************************************************************/
static ALLOC_SECTION_LDATA VK_TO_WCHARS2 aVkToWch2[] = {
//                      |         |  Shift  |
//                      |=========|=========|
  {VK_OEM_3     ,0      ,'`'      ,'~'      },
  {'1'          ,0      ,'1'      ,'!'      },
  {'2'          ,0      ,'2'      ,'@'      },
  {'3'          ,0      ,'3'      ,'#'      },
  {'4'          ,0      ,'4'      ,'$'      },
  {'5'          ,0      ,'5'      ,'%'      },
  {'6'          ,0      ,'6'      ,'^'      },
  {'7'          ,0      ,'7'      ,'&'      },
  {'8'          ,0      ,'8'      ,'*'      },
  {'9'          ,0      ,'9'      ,'('      },
  {'0'          ,0      ,'0'      ,')'      },
  {VK_OEM_MINUS ,0      ,'-'      ,'_'      },
  {VK_OEM_PLUS  ,0      ,'='      ,'+'      },
  {'Q'          ,CAPLOK ,'q'      ,'Q'      },
  {'W'          ,CAPLOK ,'w'      ,'W'      },
  {'E'          ,CAPLOK ,'e'      ,'E'      },
  {'R'          ,CAPLOK ,'r'      ,'R'      },
  {'T'          ,CAPLOK ,'t'      ,'T'      },
  {'Y'          ,CAPLOK ,'y'      ,'Y'      },
  {'I'          ,CAPLOK ,'i'      ,'I'      },
  {'P'          ,CAPLOK ,'p'      ,'P'      },
  {VK_OEM_4     ,0      ,'['      ,'{'      },
  {VK_OEM_6     ,0      ,']'      ,'}'      },
  {'D'          ,CAPLOK ,'d'      ,'D'      },
  {'F'          ,CAPLOK ,'f'      ,'F'      },
  {'G'          ,CAPLOK ,'g'      ,'G'      },
  {'H'          ,CAPLOK ,'h'      ,'H'      },
  {'J'          ,CAPLOK ,'j'      ,'J'      },
  {'K'          ,CAPLOK ,'k'      ,'K'      },
  {'L'          ,CAPLOK ,'l'      ,'L'      },
  {VK_OEM_1     ,0      ,';'      ,':'      },
  {VK_OEM_7     ,0      ,'\''     ,'\"'     },
  {VK_OEM_5     ,0      ,'\\'     ,'|'      },
  {'Z'          ,CAPLOK ,'z'      ,'Z'      },
  {'X'          ,CAPLOK ,'x'      ,'X'      },
  {'C'          ,CAPLOK ,'c'      ,'C'      },
  {'V'          ,CAPLOK ,'v'      ,'V'      },
  {'B'          ,CAPLOK ,'b'      ,'B'      },
  {'N'          ,CAPLOK ,'n'      ,'N'      },
  {'M'          ,CAPLOK ,'m'      ,'M'      },
  {VK_OEM_COMMA ,0      ,','      ,'<'      },
  {VK_OEM_PERIOD,0      ,'.'      ,'>'      },
  {VK_OEM_2     ,0      ,'/'      ,'?'      },
  {VK_DECIMAL   ,0      ,'.'      ,'.'      },
  {VK_TAB       ,0      ,'\t'     ,'\t'     },
  {VK_ADD       ,0      ,'+'      ,'+'      },
  {VK_DIVIDE    ,0      ,'/'      ,'/'      },
  {VK_MULTIPLY  ,0      ,'*'      ,'*'      },
  {VK_SUBTRACT  ,0      ,'-'      ,'-'      },
  {0            ,0      ,0        ,0        }
};

/***************************************************************************\
* aVkToWch3[] - keys with base/shift/ctrl characters
\***************************************************************************/
static ALLOC_SECTION_LDATA VK_TO_WCHARS3 aVkToWch3[] = {
//                      |         |  Shift  |  Ctrl   |
//                      |=========|=========|=========|
  {VK_OEM_102   ,0      ,'\\'     ,'|'      ,0x001c   },
  {VK_BACK      ,0      ,'\b'     ,'\b'     ,0x007f   },
  {VK_ESCAPE    ,0      ,0x001b   ,0x001b   ,0x001b   },
  {VK_RETURN    ,0      ,'\r'     ,'\r'     ,'\n'     },
  {VK_SPACE     ,0      ,' '      ,' '      ,' '      },
  {VK_CANCEL    ,0      ,0x0003   ,0x0003   ,0x0003   },
  {0            ,0      ,0        ,0        ,0        }
};

static ALLOC_SECTION_LDATA VK_TO_WCHARS1 aVkToWch1[] = {
    { VK_NUMPAD0, 0, '0' },
    { VK_NUMPAD1, 0, '1' },
    { VK_NUMPAD2, 0, '2' },
    { VK_NUMPAD3, 0, '3' },
    { VK_NUMPAD4, 0, '4' },
    { VK_NUMPAD5, 0, '5' },
    { VK_NUMPAD6, 0, '6' },
    { VK_NUMPAD7, 0, '7' },
    { VK_NUMPAD8, 0, '8' },
    { VK_NUMPAD9, 0, '9' },
    { 0,          0, '\0' }
};

/***************************************************************************\
* aVkToWch5[] - keys with AltGr / Shift+AltGr characters
*   columns: base, shift, ctrl(none), AltGr, Shift+AltGr
\***************************************************************************/
static ALLOC_SECTION_LDATA VK_TO_WCHARS5 aVkToWch5[] = {
//                     |         |  Shift  |  Ctrl   |  AltGr  |S+AltGr  |
//                     |=========|=========|=========|=========|=========|
  {'U'         ,CAPLOK ,'u'      ,'U'      ,WCH_NONE ,0x00fc   ,0x00dc   },
  {'O'         ,CAPLOK ,'o'      ,'O'      ,WCH_NONE ,0x00f6   ,0x00d6   },
  {'A'         ,CAPLOK ,'a'      ,'A'      ,WCH_NONE ,0x00e4   ,0x00c4   },
  {'S'         ,CAPLOK ,'s'      ,'S'      ,WCH_NONE ,0x00df   ,0x1e9e   },
  {0           ,0      ,0        ,0        ,0        ,0        ,0        }
};

static ALLOC_SECTION_LDATA VK_TO_WCHAR_TABLE aVkToWcharTable[] = {
    { (PVK_TO_WCHARS1)aVkToWch5, 5, sizeof(aVkToWch5[0]) },
    { (PVK_TO_WCHARS1)aVkToWch3, 3, sizeof(aVkToWch3[0]) },
    { (PVK_TO_WCHARS1)aVkToWch2, 2, sizeof(aVkToWch2[0]) },
    { (PVK_TO_WCHARS1)aVkToWch1, 1, sizeof(aVkToWch1[0]) },
    { NULL,                      0, 0                    }
};

/***************************************************************************\
* aKeyNames[], aKeyNamesExt[]
\***************************************************************************/
static ALLOC_SECTION_LDATA VSC_LPWSTR aKeyNames[] = {
    0x01, L"Esc",
    0x0e, L"Backspace",
    0x0f, L"Tab",
    0x1c, L"Enter",
    0x1d, L"Ctrl",
    0x2a, L"Shift",
    0x36, L"Right Shift",
    0x37, L"Num *",
    0x38, L"Alt",
    0x39, L"Space",
    0x3a, L"Caps Lock",
    0x3b, L"F1",
    0x3c, L"F2",
    0x3d, L"F3",
    0x3e, L"F4",
    0x3f, L"F5",
    0x40, L"F6",
    0x41, L"F7",
    0x42, L"F8",
    0x43, L"F9",
    0x44, L"F10",
    0x45, L"Pause",
    0x46, L"Scroll Lock",
    0x47, L"Num 7",
    0x48, L"Num 8",
    0x49, L"Num 9",
    0x4a, L"Num -",
    0x4b, L"Num 4",
    0x4c, L"Num 5",
    0x4d, L"Num 6",
    0x4e, L"Num +",
    0x4f, L"Num 1",
    0x50, L"Num 2",
    0x51, L"Num 3",
    0x52, L"Num 0",
    0x53, L"Num Del",
    0x54, L"Sys Req",
    0x57, L"F11",
    0x58, L"F12",
    0,    NULL
};

static ALLOC_SECTION_LDATA VSC_LPWSTR aKeyNamesExt[] = {
    0x1c, L"Num Enter",
    0x1d, L"Right Ctrl",
    0x35, L"Num /",
    0x37, L"Print Screen",
    0x38, L"Right Alt",
    0x45, L"Num Lock",
    0x46, L"Break",
    0x47, L"Home",
    0x48, L"Up",
    0x49, L"Page Up",
    0x4b, L"Left",
    0x4d, L"Right",
    0x4f, L"End",
    0x50, L"Down",
    0x51, L"Page Down",
    0x52, L"Insert",
    0x53, L"Delete",
    0x5b, L"Left Windows",
    0x5c, L"Right Windows",
    0x5d, L"Application",
    0,    NULL
};

static ALLOC_SECTION_LDATA KBDTABLES KbdTables = {
    &CharModifiers,
    aVkToWcharTable,
    NULL,
    aKeyNames,
    aKeyNamesExt,
    NULL,
    ausVK,
    sizeof(ausVK) / sizeof(ausVK[0]),
    aE0VscToVk,
    aE1VscToVk,
    0,
    0,
    0,
    NULL
};

PKBDTABLES KbdLayerDescriptor(VOID)
{
    return &KbdTables;
}
