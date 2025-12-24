import { useAuth0 } from '@auth0/auth0-react'

export function LogoutButton() {
  const { logout } = useAuth0()
  return (
    <button
      type="button"
      onClick={() => logout({ logoutParams: { returnTo: window.location.origin } })}
      className="button button--ghost"
    >
      Log out
    </button>
  )
}
