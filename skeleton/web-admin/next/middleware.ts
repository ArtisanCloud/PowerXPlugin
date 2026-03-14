import type { NextRequest } from 'next/server'
import { NextResponse } from 'next/server'

const PUBLIC_PATHS = ['/users/login', '/users/register', '/users/forgot-password']

export function middleware(request: NextRequest) {
  const { pathname, search } = request.nextUrl

  const isPublic = PUBLIC_PATHS.some((p) => pathname.startsWith(p))
  const token = request.cookies.get('access_token')?.value

  if (!isPublic && !token) {
    const redirectTo = new URL('/users/login', request.url)
    redirectTo.searchParams.set('redirect', `${pathname}${search}`)
    return NextResponse.redirect(redirectTo)
  }

  return NextResponse.next()
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
}
