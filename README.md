# filet-cloud-web
Web portal for Filet-Cloud

## Desireded Features (Still TODO)
* More content viewers:
	* md files
	* text and md file editor
	* pdf
* open content in new tab (without index) http://www.zuga.net/articles/unicode/character/1F4D6/
* select items (cart) http://www.zuga.net/articles/unicode/character/1F5C3/
	* selecting items shows a fading highlight color
* move items here (with confirmation) http://www.zuga.net/articles/unicode/character/1F69A/
* delete items (with confirmation) http://www.zuga.net/articles/unicode/character/1F5D1/
* download items http://www.zuga.net/articles/unicode/character/1F4E5/
* rename item http://www.zuga.net/articles/unicode/character/1F589/
* Share files via secure link (via making public to the pi user and thus the webserver in a PUBLIC folder). Ensure directory above is not readable but is executable. Check that is actually secure.

## Design and Engineering Philosophies

This project explores how far a software product can be pushed in terms of simplicity and minimalism, both inside and out, without losing powerful features. Window Managers are often a source of bloat, as all software tends to be. *filetwm* pushes a Window Manager to its leanest essence. It is a joy to use because it does what it needs to, and then gets out of the way. The opinions that drove the project are:

* **Complexity must justify itself**.
* Lightweight is better than heavyweight.
* Select your dependencies wisely: they are complexity, but not using them, or using the wrong ones, can lead to worse complexity.
* Powerful features are good, but simplicity and clarity are essential.
* Adding layers of simplicity, to avoid understanding something useful, only adds complexity, and is a trap for learning trivia instead of knowledge.
* Steep learning curves are dangerous, but don't just push a vertical wall deeper; learning is good, so make the incline gradual for as long as possible.
* Allow other tools to thrive - e.g: terminals don't need tabs or scrollback, that's what tmux is for.
* Fix where fixes belong - don't work around bugs in other applications, contribute to them, or make something better.
* Improvement via reduction is sometimes what a project desperately needs, because we do so tend to just add. (https://www.theregister.com/2021/04/09/people_complicate_things/)

# Thanks to, grateful forks, and contributions

We stand on the shoulders of giants. They own this, far more than I do.

* https://golang.org/
* https://developer.mozilla.org/en-US/
* https://github.com/
