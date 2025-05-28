# Web of Springs

A web of point-masses connected by springs is modeled as a graph and physics calculation.
Each springs ends accelerates its mass to reach the springs ideal length.
The force effected in this way on a springs masses is given by a factor K specific to the spring and the distance between the current positions of the two connected masses.

# Create a Web

The web of point-masses are edited by *adding a dot* and scrolling the mouse-wheel (or clicking an up/down triangle) in order to define its mass.
The mass M at a given dot is indicated by the area of the drawn dot.
When a dot is added *and only at that time* it may be *connected* to the existing dots.
In this way the connected dots forms a graph already non-cyclic
as created dot by dot, and the web can be passed as-is to the spring-model.
When a dot is connected a line is drawn to it, representing a spring.
The K factor of the spring is adjusted by the mouse-wheel (or alternatively by the up/down triangles).
Clicking a second time on a dot will *remove* last added *either dot or line*.

When the web drawn is satisfactory to the observer,
the model may be *run* by clicking the right-pointing triangle on the top left.
Clicking anywhere once running will displace the last (the selected) point-mass added.
Any other point-mass can be selected by the mouse-wheel (or clicking the arrow triangles).
Clicking the upper left double-rectangle (swapped for the triangle to run)
will go back to edit-mode so that additional dots and lines may be added,
or the existing ones removed or the last objects respective K or M value modified.

http://compctl.com/springweb-create/

# Game with Alphabet

A car is controlled by mouse position, affecting with a force to its front or back.
To the right is ahead, and letters will appear to collect among the platforms.

http://compctl.com/springweb-game/

