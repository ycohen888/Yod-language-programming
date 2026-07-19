package gpu

// OpenGL 1.1 constants used by the fixed-function backend.
const (
	cColorBufferBit     = 0x00004000
	cDepthBufferBit     = 0x00000100
	cDepthTest          = 0x0B71
	cTexture2D          = 0x0DE1
	cLighting           = 0x0B50
	cLight0             = 0x4000
	cColorMaterial      = 0x0B57
	cFrontAndBack       = 0x0408
	cAmbientAndDiffuse  = 0x1602
	cPosition           = 0x1203
	cDiffuse            = 0x1201
	cAmbient            = 0x1200
	cProjection         = 0x1701
	cModelview          = 0x1700
	cTriangles          = 0x0004
	cRGBA               = 0x1908
	cUnsignedByte       = 0x1401
	cLinear             = 0x2601
	cTextureMinFilter   = 0x2801
	cTextureMagFilter   = 0x2800
	cTextureWrapS       = 0x2802
	cTextureWrapT       = 0x2803
	cRepeat             = 0x2901
	cCullFace           = 0x0B44
	cLess               = 0x0201

	pfdDrawToWindow  = 0x00000004
	pfdSupportOpenGL = 0x00000020
	pfdDoubleBuffer  = 0x00000001
	pfdTypeRGBA      = 0
	pfdMainPlane     = 0
)
